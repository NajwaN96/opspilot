package rollout

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/opspilot/opspilot/apps/api/internal/investigate"
	"github.com/opspilot/opspilot/apps/api/internal/release"
	"github.com/opspilot/opspilot/apps/api/internal/telemetry"
)

type fakeCluster struct {
	image, version string
	canaryImage    string
	ready          bool
	desired        int
	weight         int
	promotes       int
	removes        int
}

func (f *fakeCluster) CurrentPayment(context.Context) (string, string, error) {
	return f.image, f.version, nil
}
func (f *fakeCluster) EnsureCanary(_ context.Context, image, _ string) error {
	f.canaryImage = image
	f.ready = true
	f.desired = 1
	return nil
}
func (f *fakeCluster) CanaryStatus(context.Context) (bool, int, int, string, string, error) {
	return f.ready, f.desired, f.desired, f.canaryImage, "", nil
}
func (f *fakeCluster) SetCanaryWeight(_ context.Context, _ string, weight int) error {
	f.weight = weight
	return nil
}
func (f *fakeCluster) PromoteCandidate(_ context.Context, image, version string) error {
	f.promotes++
	f.image, f.version = image, version
	f.desired = 0
	f.weight = 0
	return nil
}
func (f *fakeCluster) RemoveCanary(context.Context, string) error {
	f.removes++
	f.desired = 0
	f.weight = 0
	return nil
}
func (f *fakeCluster) RestoreBaseline(context.Context, string) error {
	f.image, f.version = release.GoodImage, release.GoodVersion
	f.desired = 0
	return nil
}

type fakeMeter struct{ samples map[string]telemetry.Snapshot }

func (f fakeMeter) VersionWindow(_ context.Context, _, _, version, _ string) (telemetry.Snapshot, error) {
	if sample, ok := f.samples[version]; ok {
		return sample, nil
	}
	return telemetry.Snapshot{Available: false}, nil
}

type memRollouts struct {
	mu     sync.Mutex
	items  map[string]Rollout
	order  []string
	events map[string][]Event
}

func newMem() *memRollouts {
	return &memRollouts{items: map[string]Rollout{}, events: map[string][]Event{}}
}
func (m *memRollouts) Create(_ context.Context, item Rollout) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, existing := range m.items {
		if activeState(existing.State) {
			return errActive
		}
	}
	m.items[item.ID] = item
	m.order = append(m.order, item.ID)
	return nil
}
func (m *memRollouts) Active(context.Context) (Rollout, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := len(m.order) - 1; i >= 0; i-- {
		item := m.items[m.order[i]]
		if activeState(item.State) {
			return item, true, nil
		}
	}
	return Rollout{}, false, nil
}
func (m *memRollouts) Get(_ context.Context, id string) (Rollout, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	item, ok := m.items[id]
	if !ok {
		return Rollout{}, errMissing
	}
	return item, nil
}
func (m *memRollouts) List(context.Context) ([]Rollout, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []Rollout
	for _, id := range m.order {
		out = append(out, m.items[id])
	}
	return out, nil
}
func (m *memRollouts) Save(_ context.Context, item Rollout) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[item.ID] = item
	return nil
}
func (m *memRollouts) AddEvent(_ context.Context, id string, event Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events[id] = append(m.events[id], event)
	return nil
}
func (m *memRollouts) Events(_ context.Context, id string) ([]Event, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]Event(nil), m.events[id]...), nil
}
func (m *memRollouts) RecordAnalysis(context.Context, string, int, Analysis) error { return nil }

var (
	errActive  = errString("active")
	errMissing = errString("missing")
)

type errString string

func (e errString) Error() string { return string(e) }

func healthy(version string) telemetry.Snapshot {
	return telemetry.Snapshot{Available: true, Requests: 30, ErrorRate: 0.01, P95: 0.04, LatencyKnown: true}
}

func TestGoodCanaryAdvancesThenPromoteIsIdempotent(t *testing.T) {
	cluster := &fakeCluster{image: release.GoodImage, version: release.GoodVersion}
	store := newMem()
	now := time.Unix(1_000, 0)
	engine := &Engine{
		Cluster: cluster, Store: store, Token: "token",
		Meter: fakeMeter{samples: map[string]telemetry.Snapshot{
			release.GoodVersion:          healthy(release.GoodVersion),
			release.CandidateGoodVersion: healthy(release.CandidateGoodVersion),
		}},
		Now: func() time.Time { return now },
	}
	view, err := engine.Start(context.Background(), "good")
	if err != nil || view.State != StatePending {
		t.Fatal(err, view.State)
	}
	if err := engine.Tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	if cluster.weight != 5 {
		t.Fatalf("weight %d", cluster.weight)
	}
	now = now.Add(MinStage + time.Second)
	if err := engine.Tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	if cluster.weight != 25 {
		t.Fatalf("advanced to %d", cluster.weight)
	}
	now = now.Add(MinStage + time.Second)
	if err := engine.Tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	now = now.Add(MinStage + time.Second)
	if err := engine.Tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	current, err := store.Get(context.Background(), view.ID)
	if err != nil || current.State != StateAwaiting || current.ProposalAction != ActionPromote || current.Weight != 50 {
		t.Fatalf("%s %s %d", current.State, current.ProposalAction, current.Weight)
	}
	if _, err := engine.Approve(context.Background(), view.ID, ActionPromote); err != nil {
		t.Fatal(err)
	}
	if _, err := engine.Approve(context.Background(), view.ID, ActionPromote); err != nil {
		t.Fatal(err)
	}
	if err := engine.Tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	if cluster.promotes != 1 {
		t.Fatalf("promotes %d", cluster.promotes)
	}
	if err := engine.Tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := engine.Tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	final, _ := store.Get(context.Background(), view.ID)
	if final.State != StateSucceeded || cluster.promotes != 1 || cluster.image != release.CandidateGoodImage {
		t.Fatalf("%s promotes %d image %s", final.State, cluster.promotes, cluster.image)
	}
}

func TestBadCanaryDoesNotAdvance(t *testing.T) {
	cluster := &fakeCluster{image: release.GoodImage, version: release.GoodVersion}
	store := newMem()
	now := time.Unix(2_000, 0)
	engine := &Engine{
		Cluster: cluster, Store: store, Token: "token",
		Meter: fakeMeter{samples: map[string]telemetry.Snapshot{
			release.CandidateBadVersion: {Available: true, Requests: 40, ErrorRate: 0.25, P95: 0.45, LatencyKnown: true},
			release.GoodVersion:         healthy(release.GoodVersion),
		}},
		Advisor: mismatchAdvisor{},
		Now:     func() time.Time { return now },
	}
	view, err := engine.Start(context.Background(), "bad")
	if err != nil {
		t.Fatal(err)
	}
	_ = engine.Tick(context.Background())
	now = now.Add(MinStage + time.Second)
	if err := engine.Tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	current, _ := store.Get(context.Background(), view.ID)
	if current.State != StateAwaiting || current.ProposalAction != ActionAbort || current.Weight != 5 || !current.AIMismatch {
		t.Fatalf("%#v", current)
	}
	if cluster.weight != 5 {
		t.Fatalf("weight moved to %d", cluster.weight)
	}
}

type mismatchAdvisor struct{}

func (mismatchAdvisor) Advise(context.Context, investigate.Snapshot) (string, string, string, bool) {
	return ActionPromote, "model wants promotion", "completed", false
}
