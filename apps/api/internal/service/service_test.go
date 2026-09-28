package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/opspilot/opspilot/apps/api/internal/executor"
	"github.com/opspilot/opspilot/apps/api/internal/repository"
	"github.com/opspilot/opspilot/apps/api/internal/repository/memory"
)

func testService(t *testing.T) *Service {
	t.Helper()
	epoch := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	store := memory.New(memory.Options{Epoch: epoch, Now: func() time.Time { return epoch }, Step: 0})
	return New(store, executor.Simulated{})
}

func TestPolicyRejectsUnapprovedAction(t *testing.T) {
	svc := testService(t)
	_, err := svc.StartRemediation(context.Background(), "INC-142", "delete-namespace")
	if !errors.Is(err, repository.ErrInvalid) {
		t.Fatalf("got %v", err)
	}
}

func TestApprovedRollbackIsIdempotent(t *testing.T) {
	svc := testService(t)
	ctx := context.Background()
	first, err := svc.StartRemediation(ctx, "INC-142", "rollback")
	if err != nil {
		t.Fatal(err)
	}
	if !first.Simulated || first.To != "v1.8.1" {
		t.Fatalf("%#v", first)
	}
	second, err := svc.StartRemediation(ctx, "INC-142", "rollback")
	if err != nil {
		t.Fatal(err)
	}
	if second.ID != first.ID {
		t.Fatalf("second %#v", second)
	}
}

func TestResetDemoDisabledByDefault(t *testing.T) {
	svc := testService(t)
	err := svc.ResetDemo(context.Background())
	if !errors.Is(err, repository.ErrForbidden) {
		t.Fatalf("got %v", err)
	}
}

func TestResetDemoClearsOnlyInMemoryRemediation(t *testing.T) {
	svc := testService(t)
	svc.SetDemoResetEnabled(true)
	ctx := context.Background()
	if _, err := svc.StartRemediation(ctx, "INC-142", "rollback"); err != nil {
		t.Fatal(err)
	}
	if err := svc.ResetDemo(ctx); err != nil {
		t.Fatal(err)
	}
	incident, err := svc.GetIncident(ctx, "INC-142")
	if err != nil {
		t.Fatal(err)
	}
	if incident.Remediation != nil || incident.Status == "resolved" {
		t.Fatalf("status %s remediation %#v", incident.Status, incident.Remediation)
	}
}

func TestRealIncidentCannotRollBack(t *testing.T) {
	svc := testService(t)
	_, err := svc.StartRemediation(context.Background(), "INC-REAL-abcdef12", "rollback")
	if !errors.Is(err, repository.ErrInvalid) {
		t.Fatalf("%v", err)
	}
}

func TestRealExperimentRequiresLab(t *testing.T) {
	svc := testService(t)
	_, err := svc.StartExperiment(context.Background(), "k8s_demo-shop_payment-api", "payment-api-degraded", 60)
	if !errors.Is(err, repository.ErrForbidden) {
		t.Fatalf("%v", err)
	}
}
