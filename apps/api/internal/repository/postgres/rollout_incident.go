package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/opspilot/opspilot/apps/api/internal/rollout"
)

const paymentDetectionService = "k8s_demo-shop_payment-api"

func (s *Store) Associated(ctx context.Context, since time.Time) (rollout.IncidentRef, bool, error) {
	var item rollout.IncidentRef
	var recovered *time.Time
	err := s.pool.QueryRow(ctx, `
		SELECT id, status, telemetry_recovered_at
		FROM incidents
		WHERE source = 'detection' AND service_id = $1 AND started_at >= $2
		ORDER BY started_at DESC LIMIT 1
	`, paymentDetectionService, since).Scan(&item.ID, &item.Status, &recovered)
	if errors.Is(err, pgx.ErrNoRows) {
		return rollout.IncidentRef{}, false, nil
	}
	if err != nil {
		return rollout.IncidentRef{}, false, err
	}
	item.Recovered = recovered != nil
	return item, true, nil
}

func (s *Store) Recoverable(ctx context.Context, since time.Time) ([]rollout.IncidentRef, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, status
		FROM incidents
		WHERE source = 'detection' AND service_id = $1 AND status <> 'resolved'
		  AND telemetry_recovered_at IS NOT NULL AND started_at >= $2
		ORDER BY started_at
	`, paymentDetectionService, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []rollout.IncidentRef
	for rows.Next() {
		var item rollout.IncidentRef
		if err := rows.Scan(&item.ID, &item.Status); err != nil {
			return nil, err
		}
		item.Recovered = true
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) ResolveRecovered(ctx context.Context, id, rolloutID, state string) (bool, error) {
	now := time.Now().UTC()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `
		UPDATE incidents SET status = 'resolved', resolved_at = $2, updated_at = $2
		WHERE id = $1 AND source = 'detection' AND service_id = $3
		  AND status <> 'resolved' AND telemetry_recovered_at IS NOT NULL
	`, id, now, paymentDetectionService)
	if err != nil {
		return false, err
	}
	if tag.RowsAffected() == 0 {
		return false, nil
	}
	verify := "Post-promotion verification healthy"
	if state == rollout.StateAborted {
		verify = "Post-abort verification healthy"
	}
	events := []struct{ title, detail string }{
		{"Rollout remediation completed", rolloutID},
		{verify, "stable workload is Ready and the candidate is not serving"},
		{"Incident resolved after verified recovery", rolloutID},
	}
	for _, event := range events {
		if _, err := tx.Exec(ctx, `
			INSERT INTO incident_events (incident_id, at, clock, title, detail, kind)
			VALUES ($1, $2, $3, $4, $5, 'recovery')
		`, id, now, now.Format("15:04:05"), event.title, event.detail); err != nil {
			return false, err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO audit_events (at, actor, incident_id, action, policy_result, approval_result, execution_status, detail)
			VALUES ($1, 'canary-controller', $2, 'incident-resolved', 'allowed', 'not-required', 'not-executed', $3)
		`, now, id, event.title+": "+event.detail); err != nil {
			return false, err
		}
	}
	return true, tx.Commit(ctx)
}
