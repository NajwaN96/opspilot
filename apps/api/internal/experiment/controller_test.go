package experiment

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/opspilot/opspilot/apps/api/internal/model"
	"github.com/opspilot/opspilot/apps/api/internal/repository"
)

type memLab struct {
	items map[string]model.Experiment
}

func (m *memLab) SaveExperiment(_ context.Context, item model.Experiment, _ map[string]string) error {
	if m.items == nil {
		m.items = map[string]model.Experiment{}
	}
	m.items[item.ID] = item
	return nil
}

func (m *memLab) GetExperiment(_ context.Context, id string) (model.Experiment, error) {
	item, ok := m.items[id]
	if !ok {
		return model.Experiment{}, repository.ErrNotFound
	}
	return item, nil
}

func (m *memLab) ListExperimentsReal(context.Context) ([]model.Experiment, error) {
	out := make([]model.Experiment, 0, len(m.items))
	for _, item := range m.items {
		out = append(out, item)
	}
	return out, nil
}

func (m *memLab) MarkExperiment(_ context.Context, id, status string) error {
	item, ok := m.items[id]
	if !ok {
		return repository.ErrNotFound
	}
	item.Status = status
	m.items[id] = item
	return nil
}

func (m *memLab) RunningExperiment(context.Context) (bool, error) {
	for _, item := range m.items {
		if item.Status == "running" {
			return true, nil
		}
	}
	return false, nil
}

type fakeFault struct {
	mode  string
	until time.Time
	fail  bool
}

func (f *fakeFault) SetPaymentFault(_ context.Context, token, mode string, until time.Time) error {
	if token == "" {
		return errors.New("missing token")
	}
	if f.fail {
		return errors.New("down")
	}
	f.mode = mode
	f.until = until
	return nil
}

func TestControllerRejectsOtherTargetsAndExpires(t *testing.T) {
	now := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	fault := &fakeFault{}
	store := &memLab{}
	lab := &Controller{
		Policy: Policy{Allow: true}, Token: "local", Fault: fault, Store: store,
		Now: func() time.Time { return now },
	}
	if _, err := lab.Start(context.Background(), "payment-api", ScenarioPaymentDegraded, 60); !errors.Is(err, repository.ErrInvalid) {
		t.Fatalf("simulated id %v", err)
	}
	if _, err := lab.Start(context.Background(), ServiceID, "delete-namespace", 60); !errors.Is(err, repository.ErrInvalid) {
		t.Fatalf("scenario %v", err)
	}
	if _, err := lab.Start(context.Background(), ServiceID, ScenarioPaymentDegraded, 300); !errors.Is(err, repository.ErrInvalid) {
		t.Fatalf("duration %v", err)
	}
	item, err := lab.Start(context.Background(), ServiceID, ScenarioPaymentDegraded, 60)
	if err != nil {
		t.Fatal(err)
	}
	if item.Simulated || fault.mode != "degraded" || !fault.until.Equal(now.Add(time.Minute)) {
		t.Fatalf("%#v mode %s", item, fault.mode)
	}
	if _, err := lab.Start(context.Background(), ServiceID, ScenarioPaymentDegraded, 60); !errors.Is(err, repository.ErrConflict) {
		t.Fatalf("second %v", err)
	}
	lab.Now = func() time.Time { return now.Add(61 * time.Second) }
	if err := lab.Expire(context.Background()); err != nil {
		t.Fatal(err)
	}
	if fault.mode != "off" {
		t.Fatalf("mode %s", fault.mode)
	}
	got, err := store.GetExperiment(context.Background(), item.ID)
	if err != nil || got.Status != "stopped" {
		t.Fatalf("%v %#v", err, got)
	}
}

func TestProductionCannotStart(t *testing.T) {
	lab := &Controller{Policy: Policy{Allow: false}, Token: "local", Fault: &fakeFault{}, Store: &memLab{}}
	_, err := lab.Start(context.Background(), ServiceID, ScenarioPaymentDegraded, 60)
	if !errors.Is(err, repository.ErrForbidden) {
		t.Fatalf("%v", err)
	}
}
