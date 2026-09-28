package telemetry

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
)

var traceIDPattern = regexp.MustCompile(`^[0-9a-f]{16,32}$`)

type TraceSummary struct {
	ID          string    `json:"id"`
	Start       time.Time `json:"start"`
	DurationMs  int       `json:"durationMs"`
	RootService string    `json:"rootService"`
	Status      string    `json:"status"`
	Spans       int       `json:"spans"`
}

type SpanView struct {
	Service    string            `json:"service"`
	Operation  string            `json:"operation"`
	DurationMs int               `json:"durationMs"`
	Status     string            `json:"status"`
	Depth      int               `json:"depth"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

type TraceDetail struct {
	TraceSummary
	Source string     `json:"source"`
	Tree   []SpanView `json:"tree"`
}

type TraceList struct {
	Source  string         `json:"source"`
	Message string         `json:"message,omitempty"`
	Traces  []TraceSummary `json:"traces"`
}

type Jaeger struct {
	Get Getter
}

func (j Jaeger) Ready(ctx context.Context) error {
	if j.Get == nil {
		return fmt.Errorf("trace backend is not configured")
	}
	code, _, err := j.Get.Get(ctx, "/api/services", nil)
	if err != nil {
		return err
	}
	if code != 200 {
		return fmt.Errorf("jaeger status %d", code)
	}
	return nil
}

func (j Jaeger) Recent(ctx context.Context, service string) (TraceList, error) {
	out := TraceList{Source: "unavailable", Message: "Telemetry unavailable", Traces: []TraceSummary{}}
	if j.Get == nil {
		return out, nil
	}
	if !namePattern.MatchString(service) {
		out.Message = "service identity is not allowed"
		return out, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	code, body, err := j.Get.Get(ctx, "/api/traces", url.Values{
		"service":  []string{service},
		"limit":    []string{"20"},
		"lookback": []string{"15m"},
	})
	if err != nil || code != 200 {
		return out, nil
	}
	traces, err := ParseTraces(body)
	if err != nil {
		return out, nil
	}
	summaries := make([]TraceSummary, 0, len(traces))
	for _, trace := range traces {
		summaries = append(summaries, trace.TraceSummary)
	}
	return TraceList{Source: "opentelemetry", Traces: summaries}, nil
}

func (j Jaeger) Trace(ctx context.Context, id string) (TraceDetail, error) {
	if !traceIDPattern.MatchString(id) {
		return TraceDetail{}, fmt.Errorf("trace id is not allowed")
	}
	if j.Get == nil {
		return TraceDetail{Source: "unavailable"}, fmt.Errorf("trace backend is unavailable")
	}
	ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	code, body, err := j.Get.Get(ctx, "/api/traces/"+id, nil)
	if err != nil || code != 200 {
		return TraceDetail{Source: "unavailable"}, fmt.Errorf("trace backend is unavailable")
	}
	traces, err := ParseTraces(body)
	if err != nil || len(traces) == 0 {
		return TraceDetail{Source: "unavailable"}, fmt.Errorf("trace not found")
	}
	traces[0].Source = "opentelemetry"
	return traces[0], nil
}

func ParseTraces(body []byte) ([]TraceDetail, error) {
	var payload struct {
		Data []rawTrace `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	out := make([]TraceDetail, 0, len(payload.Data))
	for _, trace := range payload.Data {
		if !traceIDPattern.MatchString(trace.TraceID) {
			continue
		}
		out = append(out, presentTrace(trace))
	}
	return out, nil
}

type rawTrace struct {
	TraceID   string             `json:"traceID"`
	Spans     []rawSpan          `json:"spans"`
	Processes map[string]rawProc `json:"processes"`
}

type rawProc struct {
	ServiceName string `json:"serviceName"`
}

type rawSpan struct {
	SpanID        string   `json:"spanID"`
	OperationName string   `json:"operationName"`
	References    []rawRef `json:"references"`
	StartTime     int64    `json:"startTime"`
	Duration      int64    `json:"duration"`
	Tags          []rawTag `json:"tags"`
	ProcessID     string   `json:"processID"`
}

type rawRef struct {
	RefType string `json:"refType"`
	SpanID  string `json:"spanID"`
}

type rawTag struct {
	Key   string `json:"key"`
	Type  string `json:"type"`
	Value any    `json:"value"`
}

func presentTrace(trace rawTrace) TraceDetail {
	type node struct {
		span   rawSpan
		parent string
	}
	nodes := make(map[string]node, len(trace.Spans))
	children := map[string][]string{}
	var roots []string
	for _, span := range trace.Spans {
		parent := ""
		for _, ref := range span.References {
			if ref.RefType == "CHILD_OF" && ref.SpanID != "" {
				parent = ref.SpanID
			}
		}
		nodes[span.SpanID] = node{span: span, parent: parent}
	}
	for id, item := range nodes {
		if item.parent == "" || nodes[item.parent].span.SpanID == "" {
			roots = append(roots, id)
			continue
		}
		children[item.parent] = append(children[item.parent], id)
	}
	sort.Slice(roots, func(i, j int) bool { return nodes[roots[i]].span.StartTime < nodes[roots[j]].span.StartTime })
	for parent := range children {
		sort.Slice(children[parent], func(i, j int) bool {
			return nodes[children[parent][i]].span.StartTime < nodes[children[parent][j]].span.StartTime
		})
	}
	var ordered []SpanView
	var walk func(id string, depth int)
	walk = func(id string, depth int) {
		item := nodes[id]
		ordered = append(ordered, spanView(item.span, trace.Processes, depth))
		for _, child := range children[id] {
			walk(child, depth+1)
		}
	}
	for _, id := range roots {
		walk(id, 0)
	}
	status := "ok"
	var start int64
	var end int64
	rootService := ""
	for _, span := range ordered {
		if span.Status == "error" {
			status = "error"
		}
	}
	for _, span := range trace.Spans {
		if start == 0 || span.StartTime < start {
			start = span.StartTime
		}
		if span.StartTime+span.Duration > end {
			end = span.StartTime + span.Duration
		}
	}
	if len(roots) > 0 {
		rootService = serviceName(nodes[roots[0]].span, trace.Processes)
	}
	startAt := time.UnixMicro(start).UTC()
	return TraceDetail{
		TraceSummary: TraceSummary{
			ID:          trace.TraceID,
			Start:       startAt,
			DurationMs:  int((end - start) / 1000),
			RootService: rootService,
			Status:      status,
			Spans:       len(trace.Spans),
		},
		Tree: ordered,
	}
}

func spanView(span rawSpan, processes map[string]rawProc, depth int) SpanView {
	attrs := map[string]string{}
	status := "ok"
	for _, tag := range span.Tags {
		key := strings.ToLower(tag.Key)
		if sensitive(key) {
			continue
		}
		value := fmt.Sprint(tag.Value)
		if len(value) > 80 {
			value = value[:80]
		}
		if key == "error" && value == "true" {
			status = "error"
		}
		if key == "http.status_code" {
			if value >= "500" {
				status = "error"
			}
			attrs[tag.Key] = value
			continue
		}
		if key == "otel.status_code" && strings.EqualFold(value, "ERROR") {
			status = "error"
		}
		if key == "db.system" || key == "db.operation" || key == "http.method" || key == "http.route" || key == "http.target" {
			attrs[tag.Key] = value
		}
	}
	if span.Duration > 300000 {
		attrs["slow"] = "true"
	}
	return SpanView{
		Service:    serviceName(span, processes),
		Operation:  span.OperationName,
		DurationMs: int(span.Duration / 1000),
		Status:     status,
		Depth:      depth,
		Attributes: attrs,
	}
}

func serviceName(span rawSpan, processes map[string]rawProc) string {
	if proc, ok := processes[span.ProcessID]; ok {
		return proc.ServiceName
	}
	return ""
}

func sensitive(key string) bool {
	for _, word := range []string{"authorization", "cookie", "password", "token", "secret", "header"} {
		if strings.Contains(key, word) {
			return true
		}
	}
	return false
}
