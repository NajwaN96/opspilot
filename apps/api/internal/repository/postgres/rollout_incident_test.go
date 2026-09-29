package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/opspilot/opspilot/apps/api/internal/release"
	"github.com/opspilot/opspilot/apps/api/internal/rollout"
)

func TestRolloutIncidentResolveIsNarrow(t *testing.T) {
	url := os.Getenv("OPSPILOT_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("OPSPILOT_TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	resetDatabase(t, url)
	store := openAt(t, url, "development", time.Now)
	defer store.Close()
	start := time.Now().UTC().Add(-time.Minute)
	if err := store.Create(ctx, rollout.Rollout{
		ID: "ROL-db", Service: release.Deployment, Cluster: release.Cluster, Namespace: release.Namespace,
		StableVersion: release.GoodVersion, CandidateVersion: release.CandidateBadVersion, CandidateImage: release.CandidateBadImage,
		State: rollout.StateAborted, Weight: 5, StageStarted: start, CreatedAt: start, UpdatedAt: start,
	}); err != nil {
		t.Fatal(err)
	}
	insertIncident(t, store, "INC-REAL-db", "detection", "k8s_demo-shop_payment-api", "investigating", start.Add(time.Second), true)
	insertIncident(t, store, "INC-SIM-9", "simulation", "payment-api", "investigating", start.Add(time.Second), true)
	insertIncident(t, store, "INC-REAL-old", "detection", "k8s_demo-shop_payment-api", "investigating", start.Add(-time.Hour), true)
	insertIncident(t, store, "INC-REAL-open", "detection", "k8s_demo-shop_payment-api", "investigating", start.Add(time.Second), false)

	got, ok, err := store.Associated(ctx, start)
	if err != nil || !ok || got.ID != "INC-REAL-db" {
		t.Fatalf("%s %t %v", got.ID, ok, err)
	}
	recoverable, err := store.Recoverable(ctx, start)
	if err != nil || len(recoverable) != 1 || recoverable[0].ID != "INC-REAL-db" {
		t.Fatalf("%#v %v", recoverable, err)
	}
	resolved, err := store.ResolveRecovered(ctx, "INC-REAL-db", "ROL-db", rollout.StateAborted)
	if err != nil || !resolved {
		t.Fatal(err, resolved)
	}
	again, err := store.ResolveRecovered(ctx, "INC-REAL-db", "ROL-db", rollout.StateAborted)
	if err != nil || again {
		t.Fatalf("second resolve %t %v", again, err)
	}
	var status string
	if err := store.pool.QueryRow(ctx, `SELECT status FROM incidents WHERE id = 'INC-SIM-9'`).Scan(&status); err != nil || status != "investigating" {
		t.Fatalf("unrelated %s %v", status, err)
	}
	var titles int
	if err := store.pool.QueryRow(ctx, `SELECT count(*) FROM incident_events WHERE incident_id = 'INC-REAL-db' AND title = 'Incident resolved after verified recovery'`).Scan(&titles); err != nil || titles != 1 {
		t.Fatalf("titles %d %v", titles, err)
	}
}

func insertIncident(t *testing.T, store *Store, id, source, serviceID, status string, started time.Time, recovered bool) {
	t.Helper()
	var recoveredAt *time.Time
	if recovered {
		at := started.Add(time.Second)
		recoveredAt = &at
	}
	_, err := store.pool.Exec(context.Background(), `
		INSERT INTO incidents (id, severity, service_id, service_name, title, status, started_at, summary, cluster_name, source, telemetry_recovered_at, updated_at)
		VALUES ($1, 'sev-2', $2, 'payment-api', 'Payment API reliability degradation', $3, $4, '', 'opspilot-dev', $5, $6, $4)
	`, id, serviceID, status, started, source, recoveredAt)
	if err != nil {
		t.Fatal(err)
	}
}
