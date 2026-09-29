package rollout

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/opspilot/opspilot/apps/api/internal/release"
	"github.com/opspilot/opspilot/apps/api/internal/telemetry"
)

func TestTerminalRolloutKeepsStageAndReadsLiveWeight(t *testing.T) {
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		state string
		stage int
	}{
		{StateAborted, 5},
		{StateSucceeded, 50},
	} {
		cluster := &fakeCluster{weight: 0, desired: 0, stableReady: true, image: release.GoodImage, version: release.GoodVersion}
		store := newMem()
		item := sampleRollout(tc.state, tc.stage, now)
		if err := store.Create(context.Background(), item); err != nil {
			t.Fatal(err)
		}
		engine := &Engine{Cluster: cluster, Store: store, Now: func() time.Time { return now }}
		view, err := engine.View(context.Background(), item.ID)
		if err != nil {
			t.Fatal(err)
		}
		if view.StageWeight != tc.stage || view.Weight != tc.stage || !view.LiveWeightKnown || view.LiveWeight != 0 {
			t.Fatalf("%s stage %d live %d known %t", tc.state, view.StageWeight, view.LiveWeight, view.LiveWeightKnown)
		}
		if view.CandidateActivity != "idle" {
			t.Fatalf("%s activity %s", tc.state, view.CandidateActivity)
		}
	}
}

func TestReconcileRepairsLiveWeightOnce(t *testing.T) {
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	cluster := &fakeCluster{
		image: release.GoodImage, version: release.GoodVersion, stableReady: true,
		canaryImage: release.CandidateBadImage, desired: 1, ready: true, weight: 0,
	}
	store := newMem()
	item := sampleRollout(StateRunning, 25, now)
	item.CandidateImage = release.CandidateBadImage
	item.CandidateVersion = release.CandidateBadVersion
	if err := store.Create(context.Background(), item); err != nil {
		t.Fatal(err)
	}
	meter := fakeMeter{samples: map[string]telemetry.Snapshot{
		release.CandidateBadVersion: {Available: false},
	}}
	engine := &Engine{Cluster: cluster, Store: store, Meter: meter, Now: func() time.Time { return now }}
	if err := engine.Tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	if cluster.weight != 25 {
		t.Fatalf("live weight %d", cluster.weight)
	}
	if err := engine.Tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	events, _ := store.Events(context.Background(), item.ID)
	repairs := 0
	for _, event := range events {
		if event.Title == "Reconciled live canary weight" {
			repairs++
		}
	}
	if repairs != 1 {
		t.Fatalf("repairs %d", repairs)
	}
}

func TestRecoverResolvesOnlyARecoveredPaymentIncident(t *testing.T) {
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	cluster := &fakeCluster{image: release.GoodImage, version: release.GoodVersion, weight: 0, desired: 0, stableReady: true}
	store := newMem()
	item := sampleRollout(StateAborted, 5, now)
	if err := store.Create(context.Background(), item); err != nil {
		t.Fatal(err)
	}
	incidents := &fakeIncidents{recoverable: []IncidentRef{{ID: "INC-REAL-1", Status: "investigating", Recovered: true}}}
	meter := fakeMeter{service: telemetry.Snapshot{Available: true, Requests: 40, Errors: 0, P95: 0.01, LatencyKnown: true}}
	engine := &Engine{Cluster: cluster, Store: store, Meter: meter, Incidents: incidents, Now: func() time.Time { return now }}
	if err := engine.Tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(incidents.resolved) != 1 || incidents.resolved[0] != "INC-REAL-1" {
		t.Fatalf("resolved %#v", incidents.resolved)
	}
	if err := engine.Tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(incidents.resolved) != 1 {
		t.Fatalf("second pass %#v", incidents.resolved)
	}
}

func TestMissingTelemetryDoesNotResolve(t *testing.T) {
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	cluster := &fakeCluster{image: release.GoodImage, version: release.GoodVersion, weight: 0, desired: 0, stableReady: true}
	store := newMem()
	item := sampleRollout(StateAborted, 5, now)
	if err := store.Create(context.Background(), item); err != nil {
		t.Fatal(err)
	}
	incidents := &fakeIncidents{recoverable: []IncidentRef{{ID: "INC-REAL-1", Status: "investigating", Recovered: true}}}
	engine := &Engine{Cluster: cluster, Store: store, Meter: fakeMeter{}, Incidents: incidents, Now: func() time.Time { return now }}
	if err := engine.Tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(incidents.resolved) != 0 {
		t.Fatal(incidents.resolved)
	}
}

func TestUnrecoveredIncidentIsNotResolved(t *testing.T) {
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	cluster := &fakeCluster{image: release.GoodImage, version: release.GoodVersion, weight: 0, desired: 0, stableReady: true}
	store := newMem()
	item := sampleRollout(StateAborted, 5, now)
	if err := store.Create(context.Background(), item); err != nil {
		t.Fatal(err)
	}
	incidents := &fakeIncidents{recoverable: []IncidentRef{{ID: "INC-142", Status: "investigating", Recovered: false}}}
	meter := fakeMeter{service: telemetry.Snapshot{Available: true, Requests: 40, Errors: 0, P95: 0.01}}
	engine := &Engine{Cluster: cluster, Store: store, Meter: meter, Incidents: incidents, Now: func() time.Time { return now }}
	if err := engine.Tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(incidents.resolved) != 0 {
		t.Fatal(incidents.resolved)
	}
}

func TestResetKeepsStageWeightAndClearsLiveTraffic(t *testing.T) {
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	cluster := &fakeCluster{
		image: release.CandidateBadImage, version: release.CandidateBadVersion,
		canaryImage: release.CandidateBadImage, desired: 1, ready: true, weight: 5, stableReady: true,
	}
	store := newMem()
	item := sampleRollout(StateRunning, 5, now)
	item.CandidateImage = release.CandidateBadImage
	item.CandidateVersion = release.CandidateBadVersion
	if err := store.Create(context.Background(), item); err != nil {
		t.Fatal(err)
	}
	engine := &Engine{Cluster: cluster, Store: store, Now: func() time.Time { return now }}
	if err := engine.Reset(context.Background()); err != nil {
		t.Fatal(err)
	}
	got, _ := store.Get(context.Background(), item.ID)
	if got.State != StateAborted || got.Weight != 5 {
		t.Fatalf("state %s weight %d", got.State, got.Weight)
	}
	if cluster.weight != 0 || cluster.desired != 0 || cluster.image != release.GoodImage {
		t.Fatalf("cluster image %s desired %d weight %d", cluster.image, cluster.desired, cluster.weight)
	}
	view, err := engine.View(context.Background(), item.ID)
	if err != nil || view.StageWeight != 5 || view.LiveWeight != 0 || view.CandidateActivity != "idle" {
		t.Fatalf("%+v %v", view.Rollout, err)
	}
}

func sampleRollout(state string, weight int, now time.Time) Rollout {
	return Rollout{
		ID: "ROL-test", Service: release.Deployment, Cluster: release.Cluster, Namespace: release.Namespace,
		StableVersion: release.GoodVersion, CandidateVersion: release.CandidateBadVersion, CandidateImage: release.CandidateBadImage,
		State: state, Weight: weight, StageStarted: now, CreatedAt: now, UpdatedAt: now,
	}
}

type fakeIncidents struct {
	recoverable []IncidentRef
	resolved    []string
}

func (f *fakeIncidents) Associated(context.Context, time.Time) (IncidentRef, bool, error) {
	return IncidentRef{}, false, nil
}
func (f *fakeIncidents) Recoverable(context.Context, time.Time) ([]IncidentRef, error) {
	return f.recoverable, nil
}
func (f *fakeIncidents) ResolveRecovered(_ context.Context, id, _, _ string) (bool, error) {
	for _, existing := range f.resolved {
		if existing == id {
			return false, nil
		}
	}
	f.resolved = append(f.resolved, id)
	return true, nil
}

func TestUnreadableWeightDoesNotBecomeSuccess(t *testing.T) {
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	cluster := &fakeCluster{
		image: release.GoodImage, version: release.GoodVersion, stableReady: true,
		canaryImage: release.CandidateBadImage, desired: 1, ready: true, weightErr: errors.New("unreachable"),
	}
	store := newMem()
	item := sampleRollout(StateRunning, 5, now.Add(-4*time.Minute))
	item.CandidateImage = release.CandidateBadImage
	item.StageStarted = now.Add(-4 * time.Minute)
	if err := store.Create(context.Background(), item); err != nil {
		t.Fatal(err)
	}
	engine := &Engine{Cluster: cluster, Store: store, Now: func() time.Time { return now }}
	if err := engine.Tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	got, _ := store.Get(context.Background(), item.ID)
	if got.State != StateAttention {
		t.Fatalf("state %s", got.State)
	}
}
