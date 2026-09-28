package postgres

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/opspilot/opspilot/apps/api/internal/model"
	"github.com/opspilot/opspilot/apps/api/internal/repository"
)

func TestPostgresPersistence(t *testing.T) {
	url := os.Getenv("OPSPILOT_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("OPSPILOT_TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	resetDatabase(t, url)
	epoch := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	now := epoch
	store := openAt(t, url, "development", func() time.Time { return now })
	if _, err := store.StartRemediation(ctx, demoIncident); err != nil {
		t.Fatal(err)
	}
	store.Close()

	now = epoch.Add(2 * time.Second)
	reopened := openAt(t, url, "development", func() time.Time { return now })
	defer reopened.Close()
	incident, err := reopened.GetIncident(ctx, demoIncident)
	if err != nil {
		t.Fatal(err)
	}
	if incident.Remediation == nil || incident.Status != "resolved" {
		t.Fatalf("status %s remediation %#v", incident.Status, incident.Remediation)
	}
	if len(incident.Audit) < 1 || incident.Audit[0].ApprovalResult != "approved" {
		t.Fatalf("audit %#v", incident.Audit)
	}
	var incidents int
	if err := reopened.pool.QueryRow(ctx, `SELECT count(*) FROM incidents WHERE id = $1`, demoIncident).Scan(&incidents); err != nil {
		t.Fatal(err)
	}
	if incidents != 1 {
		t.Fatalf("incident rows %d", incidents)
	}

	observed := epoch
	status := model.KubernetesStatus{Cluster: "opspilot-dev", Connectivity: "connected", KubernetesVersion: "v1.31.4", Namespaces: []string{"demo-shop"}, Mode: "local-kubernetes"}
	workload := model.Workload{
		Name: "payment-api", Namespace: "demo-shop", Kind: "Deployment", Version: "1.4.2", Image: "nginx:1.27-alpine",
		Desired: 1, Ready: 1, Status: "healthy", LastObserved: observed, ServiceID: "k8s_demo-shop_payment-api", Source: "kubernetes",
	}
	if err := reopened.ApplyObservations(ctx, status, []model.Workload{workload}, nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := reopened.ApplyObservations(ctx, status, []model.Workload{workload}, nil, nil); err != nil {
		t.Fatal(err)
	}
	var services int
	if err := reopened.pool.QueryRow(ctx, `SELECT count(*) FROM services WHERE id = 'k8s_demo-shop_payment-api'`).Scan(&services); err != nil {
		t.Fatal(err)
	}
	if services != 1 {
		t.Fatalf("discovered services %d", services)
	}

	prod, err := Open(ctx, Options{DatabaseURL: url, Env: "production", Step: 0, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	err = prod.ResetDemo(ctx)
	prod.Close()
	if !errors.Is(err, repository.ErrForbidden) {
		t.Fatalf("production reset %v", err)
	}
	var approvals int
	if err := reopened.pool.QueryRow(ctx, `SELECT count(*) FROM approvals WHERE incident_id = $1`, demoIncident).Scan(&approvals); err != nil {
		t.Fatal(err)
	}
	if approvals != 1 {
		t.Fatalf("approvals after forbidden reset %d", approvals)
	}

	if err := reopened.ResetDemo(ctx); err != nil {
		t.Fatal(err)
	}
	if err := reopened.pool.QueryRow(ctx, `SELECT count(*) FROM approvals WHERE incident_id = $1`, demoIncident).Scan(&approvals); err != nil {
		t.Fatal(err)
	}
	if approvals != 0 {
		t.Fatalf("approvals after reset %d", approvals)
	}
	if err := reopened.pool.QueryRow(ctx, `SELECT count(*) FROM services WHERE id = 'k8s_demo-shop_payment-api'`).Scan(&services); err != nil {
		t.Fatal(err)
	}
	if services != 1 {
		t.Fatal("reset deleted kubernetes service state")
	}
	restored, err := reopened.GetIncident(ctx, demoIncident)
	if err != nil {
		t.Fatal(err)
	}
	if restored.Remediation != nil || restored.Status == "resolved" {
		t.Fatalf("after reset status %s", restored.Status)
	}
	foundReset := false
	for _, record := range restored.Audit {
		if record.Action == "demo-reset" {
			foundReset = true
		}
	}
	if !foundReset {
		t.Fatalf("audit %#v", restored.Audit)
	}
}

func openAt(t *testing.T, url, env string, now func() time.Time) *Store {
	t.Helper()
	store, err := Open(context.Background(), Options{DatabaseURL: url, Env: env, Step: 0, Now: now})
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func resetDatabase(t *testing.T, url string) {
	t.Helper()
	store, err := Open(context.Background(), Options{DatabaseURL: url, Env: "development", Step: 0})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if _, err := store.pool.Exec(context.Background(), `
		TRUNCATE audit_events, approvals, remediation_executions, remediation_proposals,
		         incident_events, incidents, services, kubernetes_resources, kubernetes_events, clusters, demo_state
	`); err != nil {
		t.Fatal(err)
	}
}
