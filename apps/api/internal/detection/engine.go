package detection

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/opspilot/opspilot/apps/api/internal/model"
	"github.com/opspilot/opspilot/apps/api/internal/release"
	"github.com/opspilot/opspilot/apps/api/internal/telemetry"
)

// ClusterView is the read-only workload evidence used when an incident opens.
type ClusterView interface {
	Workload(ctx context.Context, namespace, name string) (model.Workload, error)
	Events(ctx context.Context) ([]model.ClusterEvent, error)
}

// MetricsView reads the payment-api window. A nil or failed read must not look healthy.
type MetricsView interface {
	ServiceWindow(ctx context.Context, service, namespace, window string) (telemetry.Snapshot, error)
}

// TraceView reads recent traces. Failure leaves trace evidence empty.
type TraceView interface {
	Recent(ctx context.Context, service string) (telemetry.TraceList, error)
}

// Sink persists detected incidents. It must keep one active incident per fingerprint.
type Sink interface {
	ActiveDetected(ctx context.Context, fingerprint string) (id string, recovered bool, found bool, err error)
	CreateDetected(ctx context.Context, record Record) error
	MarkRecovered(ctx context.Context, id, detail string) error
	ResumeDetected(ctx context.Context, id, detail string) error
	NoteVersion(ctx context.Context, id, version string) error
}

// Record is the bounded incident written when the rule opens.
type Record struct {
	ID             string
	Fingerprint    string
	Cluster        string
	ServiceID      string
	Facts          Facts
	Snapshot       model.Snapshot
	Evidence       model.Evidence
	Analysis       model.Analysis
	Recommendation model.Recommendation
	Events         []model.TimelineEvent
	Thresholds     map[string]any
}

type Engine struct {
	Metrics     MetricsView
	Traces      TraceView
	Cluster     ClusterView
	Sink        Sink
	ClusterName string
	Now         func() time.Time
	streak      Streak
}

func (e *Engine) now() time.Time {
	if e.Now != nil {
		return e.Now()
	}
	return time.Now()
}

// Tick evaluates one window. Missing telemetry does not open or recover an incident.
func (e *Engine) Tick(ctx context.Context) error {
	if e.Sink == nil || e.Metrics == nil {
		return nil
	}
	snap, err := e.Metrics.ServiceWindow(ctx, Service, Namespace, "1m")
	if err != nil || !snap.Available {
		snap = telemetry.Snapshot{}
	}
	decision := Evaluate(Sample{OK: snap.Available, Requests: snap.Requests, Errors: snap.Errors, P95: snap.P95, P50: snap.P50, P99: snap.P99})
	var action string
	e.streak, action = e.streak.Next(decision)
	fingerprint := Fingerprint(e.ClusterName)
	id, recovered, found, err := e.Sink.ActiveDetected(ctx, fingerprint)
	if err != nil {
		return err
	}
	if found && e.Cluster != nil {
		if workload, werr := e.Cluster.Workload(ctx, Namespace, Service); werr == nil && workload.Version != "" {
			if err := e.Sink.NoteVersion(ctx, id, workload.Version); err != nil {
				return err
			}
		}
	}
	now := e.now()
	switch {
	case found && action == "open" && recovered:
		return e.Sink.ResumeDetected(ctx, id, "The local window exceeded the threshold again.")
	case found && action == "recovered" && !recovered:
		detail := fmt.Sprintf("error rate %.1f%% and p95 %.0fms are inside the local thresholds", snap.ErrorRate*100, snap.P95*1000)
		return e.Sink.MarkRecovered(ctx, id, detail)
	case !found && action == "open":
		record := e.record(ctx, now, snap, decision)
		return e.Sink.CreateDetected(ctx, record)
	default:
		return nil
	}
}

func (e *Engine) record(ctx context.Context, now time.Time, snap telemetry.Snapshot, decision Decision) Record {
	facts := Facts{
		ErrorRate: snap.ErrorRate,
		P95:       snap.P95,
		Requests:  snap.Requests,
	}
	var traces []model.Trace
	slow := 0
	if e.Traces != nil {
		list, err := e.Traces.Recent(ctx, Service)
		if err == nil && list.Source == "opentelemetry" {
			for _, trace := range list.Traces {
				if len(traces) >= 5 {
					break
				}
				if trace.Status != "error" && trace.DurationMs < int(P95Seconds*1000) {
					continue
				}
				slow++
				traces = append(traces, model.Trace{
					ID:      trace.ID,
					TraceID: trace.ID,
					Clock:   trace.Start.UTC().Format(time.RFC3339),
					Spans: []model.Span{{
						Service:    trace.RootService,
						Name:       "trace",
						DurationMs: trace.DurationMs,
						Status:     trace.Status,
						Slow:       trace.DurationMs >= int(P95Seconds*1000),
					}},
				})
			}
		}
	}
	facts.SlowOrFailed = slow
	var kubeEvents []model.KubeEvent
	version := ""
	if e.Cluster != nil {
		if workload, err := e.Cluster.Workload(ctx, Namespace, Service); err == nil {
			facts.Ready = workload.Ready
			facts.Desired = workload.Desired
			facts.Restarts = workload.Restarts
			facts.Version = workload.Version
			version = workload.Version
		}
		if events, err := e.Cluster.Events(ctx); err == nil {
			for _, event := range events {
				if event.Namespace != Namespace || len(kubeEvents) >= 8 {
					continue
				}
				message := event.Message
				if len(message) > 180 {
					message = message[:180]
				}
				kubeEvents = append(kubeEvents, model.KubeEvent{
					Clock:   event.At.UTC().Format(time.RFC3339),
					Type:    event.Type,
					Reason:  event.Reason,
					Object:  event.Object,
					Message: message,
				})
			}
		}
	}
	finding := Diagnose(facts)
	clock := now.UTC().Format("15:04:05")
	events := []model.TimelineEvent{{
		Clock: clock, At: now, Title: "Threshold evaluation started",
		Detail: fmt.Sprintf("Rule %s evaluated a %s window", RuleID, Window), Kind: "detection",
	}}
	if snap.Requests >= MinRequests && snap.ErrorRate > ErrorRate {
		events = append(events, model.TimelineEvent{
			Clock: clock, At: now, Title: "Error rate threshold exceeded",
			Detail: fmt.Sprintf("%.1f%% over %s, threshold %.0f%%, volume %.0f", snap.ErrorRate*100, Window, ErrorRate*100, snap.Requests),
			Kind:   "metric",
		})
	}
	if snap.Requests >= MinRequests && snap.P95 > P95Seconds {
		events = append(events, model.TimelineEvent{
			Clock: clock, At: now, Title: "Latency threshold exceeded",
			Detail: fmt.Sprintf("p95 %.0fms over %s, threshold %.0fms", snap.P95*1000, Window, P95Seconds*1000),
			Kind:   "metric",
		})
	}
	events = append(events, model.TimelineEvent{
		Clock: clock, At: now, Title: "Incident opened",
		Detail: decision.Reason, Kind: "detection",
	})
	id := "INC-REAL-" + randomHex(4)
	return Record{
		ID:          id,
		Fingerprint: Fingerprint(e.ClusterName),
		Cluster:     e.ClusterName,
		ServiceID:   "k8s_demo-shop_payment-api",
		Facts:       facts,
		Snapshot: model.Snapshot{
			Availability: snap.Availability * 100,
			P95LatencyMs: int(snap.P95 * 1000),
			ErrorRate:    snap.ErrorRate * 100,
			Status:       "degraded",
			Version:      version,
		},
		Evidence: model.Evidence{
			Metrics: []model.MetricPoint{{
				Clock:        clock,
				P95LatencyMs: int(snap.P95 * 1000),
				ErrorRate:    snap.ErrorRate * 100,
				RequestRate:  snap.RequestRate,
			}},
			Traces:           traces,
			KubernetesEvents: kubeEvents,
		},
		Analysis: model.Analysis{
			Simulated:     false,
			Cause:         finding.LikelyCause,
			LikelyCause:   finding.LikelyCause,
			Supporting:    finding.Supporting,
			Contradicting: finding.Contradicting,
			Evidence:      finding.Supporting,
		},
		Recommendation: release.Propose(version),
		Events:         events,
		Thresholds: map[string]any{
			"rule":         RuleID,
			"minRequests":  MinRequests,
			"errorRate":    ErrorRate,
			"p95Seconds":   P95Seconds,
			"window":       Window.String(),
			"sustain":      Sustain,
			"recover":      Recover,
			"evalInterval": EvalInterval.String(),
		},
	}
}

func randomHex(n int) string {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "00000000"
	}
	return hex.EncodeToString(buf)
}
