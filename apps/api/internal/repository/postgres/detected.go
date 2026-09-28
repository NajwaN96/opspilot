package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/opspilot/opspilot/apps/api/internal/detection"
	"github.com/opspilot/opspilot/apps/api/internal/model"
	"github.com/opspilot/opspilot/apps/api/internal/repository"
)

type storedDetection struct {
	Facts          detection.Facts      `json:"facts"`
	Snapshot       model.Snapshot       `json:"snapshot"`
	Evidence       model.Evidence       `json:"evidence"`
	Analysis       model.Analysis       `json:"analysis"`
	Recommendation model.Recommendation `json:"recommendation"`
	Thresholds     map[string]any       `json:"thresholds"`
}

func (s *Store) ActiveDetected(ctx context.Context, fingerprint string) (string, bool, bool, error) {
	var id string
	var recovered *time.Time
	err := s.pool.QueryRow(ctx, `
		SELECT id, telemetry_recovered_at
		FROM incidents
		WHERE fingerprint = $1 AND status <> 'resolved'
		ORDER BY started_at DESC
		LIMIT 1
	`, fingerprint).Scan(&id, &recovered)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, false, nil
	}
	if err != nil {
		return "", false, false, err
	}
	return id, recovered != nil, true, nil
}

func (s *Store) CreateDetected(ctx context.Context, record detection.Record) error {
	blob, err := json.Marshal(storedDetection{
		Facts: record.Facts, Snapshot: record.Snapshot, Evidence: record.Evidence,
		Analysis: record.Analysis, Recommendation: record.Recommendation, Thresholds: record.Thresholds,
	})
	if err != nil {
		return err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	_, err = tx.Exec(ctx, `
		INSERT INTO incidents (
			id, severity, service_id, service_name, title, status, started_at, summary, cluster_name,
			source, metadata, fingerprint, rule_id, updated_at
		) VALUES (
			$1, 'sev-2', $2, 'payment-api', 'Payment API reliability degradation', 'investigating', $3, $4, $5,
			'detection', $6, $7, $8, $3
		)
	`, record.ID, record.ServiceID, record.Events[len(record.Events)-1].At, record.Analysis.LikelyCause, record.Cluster, blob, record.Fingerprint, detection.RuleID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil
		}
		return err
	}
	for _, event := range record.Events {
		if _, err := tx.Exec(ctx, `
			INSERT INTO incident_events (incident_id, at, clock, title, detail, kind)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, record.ID, event.At, event.Clock, event.Title, event.Detail, event.Kind); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events (at, actor, incident_id, action, policy_result, approval_result, execution_status, detail)
		VALUES ($1, $2, $3, 'incident-opened', 'detection-rule', 'not-required', 'not-executed', $4)
	`, record.Events[0].At, "detection-engine", record.ID, detection.RuleID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) MarkRecovered(ctx context.Context, id, detail string) error {
	now := time.Now().UTC()
	tag, err := s.pool.Exec(ctx, `
		UPDATE incidents SET telemetry_recovered_at = $2, updated_at = $2
		WHERE id = $1 AND telemetry_recovered_at IS NULL AND status <> 'resolved'
	`, id, now)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return nil
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO incident_events (incident_id, at, clock, title, detail, kind)
		VALUES ($1, $2, $3, 'Telemetry recovered', $4, 'recovery')
	`, id, now, now.Format("15:04:05"), detail)
	return err
}

func (s *Store) ResumeDetected(ctx context.Context, id, detail string) error {
	now := time.Now().UTC()
	tag, err := s.pool.Exec(ctx, `
		UPDATE incidents SET telemetry_recovered_at = NULL, updated_at = $2
		WHERE id = $1 AND telemetry_recovered_at IS NOT NULL AND status <> 'resolved'
	`, id, now)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return nil
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO incident_events (incident_id, at, clock, title, detail, kind)
		VALUES ($1, $2, $3, 'Degradation resumed', $4, 'detection')
	`, id, now, now.Format("15:04:05"), detail)
	return err
}

func (s *Store) listDetected(ctx context.Context) ([]model.Incident, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id FROM incidents WHERE source = 'detection' ORDER BY started_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []model.Incident
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, id := range ids {
		item, err := s.getDetected(ctx, id)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *Store) getDetected(ctx context.Context, id string) (model.Incident, error) {
	var (
		item      model.Incident
		started   time.Time
		resolved  *time.Time
		recovered *time.Time
		summary   string
		metadata  []byte
		status    string
		cluster   string
		serviceID string
	)
	err := s.pool.QueryRow(ctx, `
		SELECT id, severity, service_id, service_name, title, status, started_at, resolved_at, summary,
		       cluster_name, metadata, rule_id, telemetry_recovered_at
		FROM incidents WHERE id = $1 AND source = 'detection'
	`, id).Scan(&item.ID, &item.Severity, &serviceID, &item.ServiceName, &item.Title, &status, &started, &resolved, &summary, &cluster, &metadata, &item.RuleID, &recovered)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Incident{}, fmt.Errorf("%w: incident %s", repository.ErrNotFound, id)
	}
	if err != nil {
		return model.Incident{}, err
	}
	var blob storedDetection
	if len(metadata) > 0 {
		if err := json.Unmarshal(metadata, &blob); err != nil {
			return model.Incident{}, err
		}
	}
	finding := detection.Diagnose(blob.Facts)
	blob.Analysis.Simulated = false
	blob.Analysis.Confidence = 0
	blob.Analysis.LikelyCause = finding.LikelyCause
	blob.Analysis.Cause = finding.LikelyCause
	blob.Analysis.Supporting = finding.Supporting
	blob.Analysis.Contradicting = finding.Contradicting
	blob.Analysis.Evidence = finding.Supporting
	blob.Recommendation.Allowed = false
	events, err := s.detectedEvents(ctx, id)
	if err != nil {
		return model.Incident{}, err
	}
	audit, err := s.listAudit(ctx, id)
	if err != nil {
		return model.Incident{}, err
	}
	end := time.Now()
	if resolved != nil {
		end = *resolved
	}
	item.ServiceID = serviceID
	item.Status = status
	item.StartedAt = started
	item.ResolvedAt = resolved
	item.DurationSec = int(end.Sub(started).Seconds())
	item.Summary = summary
	item.Cluster = cluster
	item.Timeline = events
	item.Evidence = &blob.Evidence
	item.Analysis = &blob.Analysis
	item.Recommendation = &blob.Recommendation
	item.Snapshot = blob.Snapshot
	item.Audit = audit
	item.Origin = "detection-engine"
	item.MetricsSource = "Prometheus"
	item.TraceSource = "OpenTelemetry"
	item.KubernetesSource = cluster
	item.TelemetryRecoveredAt = recovered
	item.Thresholds = blob.Thresholds
	return item, nil
}

func (s *Store) detectedEvents(ctx context.Context, id string) ([]model.TimelineEvent, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT at, clock, title, detail, kind FROM incident_events WHERE incident_id = $1 ORDER BY at, id
	`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var events []model.TimelineEvent
	for rows.Next() {
		var event model.TimelineEvent
		if err := rows.Scan(&event.At, &event.Clock, &event.Title, &event.Detail, &event.Kind); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}
