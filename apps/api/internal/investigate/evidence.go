package investigate

import (
	"fmt"
	"strings"
	"time"

	"github.com/opspilot/opspilot/apps/api/internal/detection"
	"github.com/opspilot/opspilot/apps/api/internal/model"
	"github.com/opspilot/opspilot/apps/api/internal/release"
	"github.com/opspilot/opspilot/apps/api/internal/telemetry"
)

const (
	maxTraces = 4
	maxSpans  = 6
	maxEvents = 6
	maxPods   = 4
)

// Subject is the incident the evidence builder is allowed to see.
type Subject struct {
	ID         string
	Title      string
	Service    string
	Namespace  string
	Cluster    string
	Severity   string
	Status     string
	Rule       string
	Started    time.Time
	Snapshot   model.Snapshot
	Analysis   model.Analysis
	Thresholds map[string]any
	Facts      detection.Facts
}

// Observations are read-only inputs. They are not a cluster client.
type Observations struct {
	Now      time.Time
	Metrics  telemetry.Snapshot
	SLO      telemetry.Snapshot
	Traces   []telemetry.TraceDetail
	Workload model.Workload
	Pods     []model.Pod
	Events   []model.ClusterEvent
}

// Build creates a bounded snapshot. Trace IDs that do not appear here cannot be cited.
func Build(subject Subject, obs Observations) Snapshot {
	now := obs.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	var items []Item
	var omitted []string
	items = append(items, Item{
		ID: "RULE-001", Source: "Detection Engine", Kind: "rule", At: subject.Started,
		Title: subject.Rule,
		Body: fmt.Sprintf("incident %s service %s namespace %s cluster %s severity %s status %s title %s",
			subject.ID, subject.Service, subject.Namespace, subject.Cluster, subject.Severity, subject.Status, subject.Title),
	})
	if len(subject.Thresholds) > 0 {
		items = append(items, Item{
			ID: "RULE-002", Source: "Detection Engine", Kind: "thresholds", At: subject.Started,
			Title: "Thresholds",
			Body:  fmt.Sprintf("%v", subject.Thresholds),
		})
	}
	items = append(items,
		metricItem("METRIC-VOLUME-001", "Request volume", obs.Metrics, fmt.Sprintf("requests %.0f rate %.2f/s", obs.Metrics.Requests, obs.Metrics.RequestRate)),
		metricItem("METRIC-ERROR-001", "Error rate", obs.Metrics, fmt.Sprintf("error rate %.1f%% errors %.0f", obs.Metrics.ErrorRate*100, obs.Metrics.Errors)),
		metricItem("METRIC-LATENCY-001", "Latency", obs.Metrics, fmt.Sprintf("p50 %.0fms p95 %.0fms p99 %.0fms", obs.Metrics.P50*1000, obs.Metrics.P95*1000, obs.Metrics.P99*1000)),
	)
	if obs.SLO.Available || obs.SLO.Requests > 0 {
		slo := detection.ComputeSLO(obs.SLO.Requests, obs.SLO.Errors)
		items = append(items, Item{
			ID: "METRIC-SLO-001", Source: "Prometheus", Kind: "slo", At: obs.SLO.Updated,
			Title: "Local 15m availability",
			Body: fmt.Sprintf("target %.1f%% availability %.2f%% burn %.2f sufficient %t window %s. This is not a 30-day SLO.",
				slo.Target*100, slo.Availability*100, slo.BurnRate, slo.Sufficient, slo.Window),
		})
	}
	traces := obs.Traces
	if len(traces) > maxTraces {
		omitted = append(omitted, fmt.Sprintf("traces omitted %d", len(traces)-maxTraces))
		traces = traces[:maxTraces]
	}
	spanN := 1
	for i, trace := range traces {
		spans := trace.Tree
		if len(spans) > maxSpans {
			omitted = append(omitted, fmt.Sprintf("spans omitted on %s", trace.ID))
			spans = spans[:maxSpans]
		}
		var lines []string
		for _, span := range spans {
			lines = append(lines, fmt.Sprintf("SPAN-%03d %s %s %dms %s attrs %s", spanN, span.Service, span.Operation, span.DurationMs, span.Status, attributeText(span.Attributes)))
			spanN++
		}
		items = append(items, Item{
			ID: fmt.Sprintf("TRACE-%03d", i+1), Source: "OpenTelemetry / Jaeger", Kind: "trace", At: trace.Start,
			Title: trace.ID,
			Body:  fmt.Sprintf("trace %s root %s status %s duration %dms spans %s", trace.ID, trace.RootService, trace.Status, trace.DurationMs, strings.Join(lines, "; ")),
		})
	}
	workload := obs.Workload
	items = append(items, Item{
		ID: "K8S-DEPLOYMENT-001", Source: "Kubernetes API", Kind: "deployment", At: workload.LastObserved,
		Title: workload.Name,
		Body: fmt.Sprintf("namespace %s deployment %s image %s version %s desired %d ready %d restarts %d status %s labels %s",
			subject.Namespace, workload.Name, workload.Image, workload.Version, workload.Desired, workload.Ready, workload.Restarts, workload.Status, labelText(workload.Labels)),
	})
	pods := obs.Pods
	if len(pods) > maxPods {
		omitted = append(omitted, fmt.Sprintf("pods omitted %d", len(pods)-maxPods))
		pods = pods[:maxPods]
	}
	if len(pods) > 0 {
		var lines []string
		for _, pod := range pods {
			lines = append(lines, fmt.Sprintf("%s status %s ready %s restarts %d", pod.Name, pod.Status, pod.Ready, pod.Restarts))
		}
		items = append(items, Item{
			ID: "K8S-POD-001", Source: "Kubernetes API", Kind: "pod", At: now,
			Title: "payment-api pods", Body: strings.Join(lines, "; "),
		})
	}
	events := obs.Events
	if len(events) > maxEvents {
		omitted = append(omitted, fmt.Sprintf("events omitted %d", len(events)-maxEvents))
		events = events[:maxEvents]
	}
	if len(events) > 0 {
		var lines []string
		for _, event := range events {
			lines = append(lines, fmt.Sprintf("%s %s %s %s", event.Type, event.Reason, event.Object, clip(event.Message, 160)))
		}
		items = append(items, Item{
			ID: "K8S-EVENTS-001", Source: "Kubernetes API", Kind: "event", At: now,
			Title: "Recent events", Body: strings.Join(lines, "; "),
		})
	}
	previous := release.GoodVersion
	if workload.Version == release.GoodVersion {
		previous = ""
	}
	items = append(items, Item{
		ID: "CHANGE-001", Source: "Deployment Metadata", Kind: "change", At: workload.LastObserved,
		Title: "payment-api release",
		Body: fmt.Sprintf("current version %s image %s previous known good %s known bad %s",
			workload.Version, workload.Image, previous, release.BadVersion),
	})
	items = append(items, Item{
		ID: "DIAG-001", Source: "Deterministic Diagnosis", Kind: "diagnosis", At: now,
		Title: "Deterministic findings",
		Body: fmt.Sprintf("likely %s supporting %s contradicting %s",
			subject.Analysis.LikelyCause, strings.Join(subject.Analysis.Supporting, "; "), strings.Join(subject.Analysis.Contradicting, "; ")),
	})
	excerpt := RunbookExcerpt(subject.Rule, subject.Service)
	items = append(items, Item{
		ID: "RUNBOOK-001", Source: "Runbook", Kind: "runbook", At: now,
		Title: "payment-api runbooks", Body: excerpt,
	})
	items = SanitizeItems(items)
	return Snapshot{
		IncidentID: subject.ID,
		CreatedAt:  now,
		Items:      items,
		Omitted:    omitted,
		TraceCount: len(traces),
	}
}

func attributeText(values map[string]string) string {
	if len(values) == 0 {
		return "none"
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sortStrings(keys)
	var parts []string
	for _, key := range keys {
		if len(parts) == 4 {
			break
		}
		parts = append(parts, key+"="+clip(values[key], 80))
	}
	return strings.Join(parts, ",")
}

func labelText(values map[string]string) string {
	if len(values) == 0 {
		return "none"
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sortStrings(keys)
	var parts []string
	for _, key := range keys {
		if len(parts) == 6 {
			break
		}
		parts = append(parts, key+"="+clip(values[key], 80))
	}
	return strings.Join(parts, ",")
}

func sortStrings(values []string) {
	for i := 1; i < len(values); i++ {
		for j := i; j > 0 && values[j] < values[j-1]; j-- {
			values[j], values[j-1] = values[j-1], values[j]
		}
	}
}

func metricItem(id, title string, snap telemetry.Snapshot, body string) Item {
	source := "Prometheus"
	if !snap.Available {
		body = "prometheus window unavailable. " + body
	}
	return Item{ID: id, Source: source, Kind: "metric", At: snap.Updated, Title: title, Body: body}
}

// SelectTraces keeps a few failed traces and a few slow traces.
func SelectTraces(list telemetry.TraceList) []string {
	var failed, slow []string
	for _, trace := range list.Traces {
		if trace.Status == "error" && len(failed) < 2 {
			failed = append(failed, trace.ID)
			continue
		}
		if trace.DurationMs >= 300 && len(slow) < 2 {
			slow = append(slow, trace.ID)
		}
	}
	return append(failed, slow...)
}
