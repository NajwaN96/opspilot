package detection

import (
	"context"
	"testing"
	"time"

	"github.com/opspilot/opspilot/apps/api/internal/model"
	"github.com/opspilot/opspilot/apps/api/internal/telemetry"
)

type fakeMetrics struct{ snap telemetry.Snapshot }

func (f fakeMetrics) ServiceWindow(context.Context, string, string, string) (telemetry.Snapshot, error) {
	return f.snap, nil
}

type fakeTraces struct{ list telemetry.TraceList }

func (f fakeTraces) Recent(context.Context, string) (telemetry.TraceList, error) { return f.list, nil }

type fakeCluster struct{}

func (fakeCluster) Workload(context.Context, string, string) (model.Workload, error) {
	return model.Workload{Name: Service, Namespace: Namespace, Version: "0.3.0", Desired: 1, Ready: 1}, nil
}

func (fakeCluster) Events(context.Context) ([]model.ClusterEvent, error) { return nil, nil }

type fakeSink struct {
	id         string
	recovered  bool
	found      bool
	created    int
	recoveredN int
}

func (f *fakeSink) ActiveDetected(context.Context, string) (string, bool, bool, error) {
	return f.id, f.recovered, f.found, nil
}

func (f *fakeSink) CreateDetected(_ context.Context, record Record) error {
	f.created++
	f.found = true
	f.id = record.ID
	f.recovered = false
	if record.Analysis.LikelyCause == "" || record.Recommendation.Allowed {
		return errAllowed
	}
	return nil
}

func (f *fakeSink) MarkRecovered(context.Context, string, string) error {
	f.recovered = true
	f.recoveredN++
	return nil
}

func (f *fakeSink) ResumeDetected(context.Context, string, string) error {
	f.recovered = false
	return nil
}

var errAllowed = errorString("recommendation must not be executable")

type errorString string

func (e errorString) Error() string { return string(e) }

func TestEngineOpensOnceThenRecovers(t *testing.T) {
	sink := &fakeSink{}
	bad := telemetry.Snapshot{Available: true, Requests: 80, Errors: 20, ErrorRate: 0.25, P95: 0.5, RequestRate: 1.3}
	engine := &Engine{
		Metrics: fakeMetrics{snap: bad}, Traces: fakeTraces{list: telemetry.TraceList{Source: "opentelemetry", Traces: []telemetry.TraceSummary{{
			ID: "0123456789abcdef", Status: "error", DurationMs: 500, RootService: "storefront",
		}}}},
		Cluster: fakeCluster{}, Sink: sink, ClusterName: "opspilot-dev", Now: func() time.Time { return time.Unix(100, 0) },
	}
	if err := engine.Tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	if sink.created != 0 {
		t.Fatal("first breach must wait for the sustain window")
	}
	if err := engine.Tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	if sink.created != 1 {
		t.Fatalf("created %d", sink.created)
	}
	if err := engine.Tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	if sink.created != 1 {
		t.Fatal("duplicate incident")
	}
	engine.Metrics = fakeMetrics{snap: telemetry.Snapshot{Available: true, Requests: 80, Errors: 0, P95: 0.05}}
	if err := engine.Tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	if sink.recoveredN != 0 {
		t.Fatal("recovery must also be sustained")
	}
	if err := engine.Tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	if sink.recoveredN != 1 || !sink.recovered {
		t.Fatalf("recovered %#v", sink)
	}
}

func TestEngineIgnoresMissingTelemetry(t *testing.T) {
	sink := &fakeSink{}
	engine := &Engine{Metrics: fakeMetrics{}, Sink: sink, ClusterName: "opspilot-dev"}
	if err := engine.Tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := engine.Tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	if sink.created != 0 {
		t.Fatal("missing telemetry opened an incident")
	}
}
