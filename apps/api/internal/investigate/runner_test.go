package investigate

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/opspilot/opspilot/apps/api/internal/model"
	"github.com/opspilot/opspilot/apps/api/internal/release"
)

type memStore struct {
	mu      sync.Mutex
	subject Subject
	snaps   []Snapshot
	rows    []Record
	events  []string
}

func (m *memStore) OpenSubjects(context.Context) ([]Subject, error)  { return []Subject{m.subject}, nil }
func (m *memStore) Subject(context.Context, string) (Subject, error) { return m.subject, nil }
func (m *memStore) LatestSnapshot(context.Context, string) (Snapshot, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.snaps) == 0 {
		return Snapshot{}, false, nil
	}
	return m.snaps[len(m.snaps)-1], true, nil
}
func (m *memStore) SaveSnapshot(_ context.Context, snapshot Snapshot) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.snaps = append(m.snaps, snapshot)
	return nil
}
func (m *memStore) LatestInvestigation(context.Context, string) (Record, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.rows) == 0 {
		return Record{}, false, nil
	}
	return m.rows[len(m.rows)-1], true, nil
}
func (m *memStore) BeginInvestigation(_ context.Context, record Record) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, row := range m.rows {
		if row.IncidentID == record.IncidentID && row.SnapshotVersion == record.SnapshotVersion && row.Status == StatusInvestigating {
			return false, nil
		}
	}
	m.rows = append(m.rows, record)
	return true, nil
}
func (m *memStore) FinishInvestigation(_ context.Context, record Record) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.rows {
		if m.rows[i].ID == record.ID {
			m.rows[i] = record
		}
	}
	return nil
}
func (m *memStore) AppendIncidentEvent(_ context.Context, _, title, _, _ string) error {
	m.events = append(m.events, title)
	return nil
}
func (m *memStore) MergeTraces(context.Context, string, []model.Trace) error { return nil }
func (m *memStore) SnapshotByID(_ context.Context, id string) (Snapshot, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, snap := range m.snaps {
		if snap.ID == id {
			return snap, true, nil
		}
	}
	return Snapshot{}, false, nil
}

func TestDuplicateInvestigationDoesNotCallTheProviderTwice(t *testing.T) {
	calls := 0
	store := &memStore{subject: Subject{
		ID: "INC-REAL-test", Service: "payment-api", Namespace: "demo-shop", Rule: "PAYMENT_API_RELIABILITY_DEGRADATION",
		Started: time.Now().Add(-time.Minute), Analysis: model.Analysis{LikelyCause: "known bad release"},
	}}
	runner := &Runner{
		Provider: Fixture{Output: Output{
			Summary:           "fixture",
			RecommendedAction: Action{ActionType: ActionRollback, Reason: "fixture", EvidenceIDs: []string{"CHANGE-001"}},
			LikelyCauses:      []Cause{{Cause: "fixture", Confidence: 0.4, SupportingEvidenceIDs: []string{"CHANGE-001"}}},
		}},
		Store: store,
		Now:   func() time.Time { return time.Unix(200, 0) },
	}
	runner.Provider = countingFixture{inner: runner.Provider, calls: &calls}
	if err := runner.Consider(context.Background(), store.subject.ID, true); err != nil {
		t.Fatal(err)
	}
	if err := runner.Consider(context.Background(), store.subject.ID, true); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("provider calls %d", calls)
	}
	view, err := runner.View(context.Background(), store.subject.ID)
	if err != nil {
		t.Fatal(err)
	}
	if view.Real || view.Provider != "fixture" || view.RecommendedAction.ActionType != ActionRollback {
		t.Fatalf("%#v", view)
	}
}

type countingFixture struct {
	inner Provider
	calls *int
}

func (c countingFixture) Name() string                      { return c.inner.Name() }
func (c countingFixture) Model() string                     { return c.inner.Model() }
func (c countingFixture) Real() bool                        { return c.inner.Real() }
func (c countingFixture) Status(ctx context.Context) string { return c.inner.Status(ctx) }
func (c countingFixture) Investigate(ctx context.Context, snapshot Snapshot) (Output, Usage, error) {
	*c.calls++
	return c.inner.Investigate(ctx, snapshot)
}

func TestDisabledProviderDoesNotClaimARealInvestigation(t *testing.T) {
	store := &memStore{subject: Subject{ID: "INC-REAL-test", Service: "payment-api", Started: time.Unix(10, 0)}}
	runner := &Runner{Provider: Disabled{}, Store: store, Now: func() time.Time { return time.Unix(200, 0) }}
	if err := runner.Consider(context.Background(), store.subject.ID, true); err == nil {
		t.Fatal("disabled provider returned success")
	}
	view, err := runner.View(context.Background(), store.subject.ID)
	if err != nil {
		t.Fatal(err)
	}
	if view.Real || view.Status != StatusFailed {
		t.Fatalf("%#v", view)
	}
}

func TestAIRollbackRecommendationDoesNotAuthorizeTheProposal(t *testing.T) {
	snap := sampleSnapshot()
	out, result := Validate(snap, []byte(`{
		"summary":"payment-api 1.5.0-bad matches the evidence",
		"likely_causes":[{"cause":"application regression","confidence":0.6,"supporting_evidence_ids":["CHANGE-001"],"contradicting_evidence_ids":[]}],
		"observations":[],
		"recommended_action":{"action_type":"ROLLBACK_PAYMENT_API","reason":"known bad release","evidence_ids":["CHANGE-001"]},
		"missing_information":[],
		"risk_notes":[]
	}`))
	if !result.Accepted || out.RecommendedAction.ActionType != ActionRollback {
		t.Fatalf("%#v", result)
	}
	if out.RecommendedAction.ActionType == release.Action {
		t.Fatal("semantic recommendation used the executor action name")
	}
	denied := release.Propose(release.GoodVersion)
	if denied.Allowed || denied.Action == release.Action {
		t.Fatalf("healthy version became executable: %#v", denied)
	}
	if err := release.Validate(out.RecommendedAction.ActionType, "payment-api", "demo-shop", "1.5.0-bad", "1.4.2"); err == nil {
		t.Fatal("AI action string passed executor validation")
	}
}

func TestProviderOutageIsNotRelabeledAsFixture(t *testing.T) {
	provider := OpenAI{
		APIKey: "test-key",
		Client: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusServiceUnavailable, Body: io.NopCloser(strings.NewReader(`{"error":{"type":"server_error","code":"overloaded"}}`)), Header: make(http.Header)}, nil
		})},
	}
	_, _, err := provider.Investigate(context.Background(), Snapshot{IncidentID: "INC"})
	if err == nil || !provider.Real() || provider.Name() != "openai" || !strings.Contains(err.Error(), "unavailable") {
		t.Fatalf("%v", err)
	}
}
