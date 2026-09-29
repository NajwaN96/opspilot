package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/opspilot/opspilot/apps/api/internal/detection"
	"github.com/opspilot/opspilot/apps/api/internal/investigate"
	"github.com/opspilot/opspilot/apps/api/internal/model"
	"github.com/opspilot/opspilot/apps/api/internal/repository"
)

func (s *Store) OpenSubjects(ctx context.Context) ([]investigate.Subject, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id FROM incidents WHERE source = 'detection' AND status <> 'resolved' ORDER BY started_at DESC LIMIT 5
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
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
	var out []investigate.Subject
	for _, id := range ids {
		subject, err := s.Subject(ctx, id)
		if err != nil {
			return nil, err
		}
		out = append(out, subject)
	}
	return out, nil
}

func (s *Store) Subject(ctx context.Context, id string) (investigate.Subject, error) {
	var (
		subject  investigate.Subject
		started  time.Time
		metadata []byte
		summary  string
	)
	err := s.pool.QueryRow(ctx, `
		SELECT id, title, service_name, severity, status, started_at, cluster_name, COALESCE(rule_id, ''), summary, metadata
		FROM incidents WHERE id = $1 AND source = 'detection'
	`, id).Scan(&subject.ID, &subject.Title, &subject.Service, &subject.Severity, &subject.Status, &started, &subject.Cluster, &subject.Rule, &summary, &metadata)
	if errors.Is(err, pgx.ErrNoRows) {
		return investigate.Subject{}, fmtNotFound(id)
	}
	if err != nil {
		return investigate.Subject{}, err
	}
	subject.Started = started
	subject.Namespace = "demo-shop"
	var blob storedDetection
	if len(metadata) > 0 {
		_ = json.Unmarshal(metadata, &blob)
	}
	subject.Facts = blob.Facts
	subject.Snapshot = blob.Snapshot
	subject.Analysis = blob.Analysis
	subject.Thresholds = blob.Thresholds
	if subject.Analysis.LikelyCause == "" {
		finding := detection.Diagnose(blob.Facts)
		subject.Analysis.LikelyCause = finding.LikelyCause
		subject.Analysis.Supporting = finding.Supporting
		subject.Analysis.Contradicting = finding.Contradicting
	}
	if subject.Service == "" {
		subject.Service = "payment-api"
	}
	return subject, nil
}

func fmtNotFound(id string) error {
	return errors.Join(repository.ErrNotFound, errors.New(id))
}

func (s *Store) LatestSnapshot(ctx context.Context, incidentID string) (investigate.Snapshot, bool, error) {
	var payload []byte
	err := s.pool.QueryRow(ctx, `
		SELECT payload FROM incident_evidence_snapshots WHERE incident_id = $1 ORDER BY version DESC LIMIT 1
	`, incidentID).Scan(&payload)
	if errors.Is(err, pgx.ErrNoRows) {
		return investigate.Snapshot{}, false, nil
	}
	if err != nil {
		return investigate.Snapshot{}, false, err
	}
	var snapshot investigate.Snapshot
	if err := json.Unmarshal(payload, &snapshot); err != nil {
		return investigate.Snapshot{}, false, err
	}
	return snapshot, true, nil
}

func (s *Store) SnapshotByID(ctx context.Context, id string) (investigate.Snapshot, bool, error) {
	var payload []byte
	err := s.pool.QueryRow(ctx, `SELECT payload FROM incident_evidence_snapshots WHERE id = $1`, id).Scan(&payload)
	if errors.Is(err, pgx.ErrNoRows) {
		return investigate.Snapshot{}, false, nil
	}
	if err != nil {
		return investigate.Snapshot{}, false, err
	}
	var snapshot investigate.Snapshot
	if err := json.Unmarshal(payload, &snapshot); err != nil {
		return investigate.Snapshot{}, false, err
	}
	return snapshot, true, nil
}

func (s *Store) SaveSnapshot(ctx context.Context, snapshot investigate.Snapshot) error {
	payload, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO incident_evidence_snapshots (id, incident_id, version, payload)
		VALUES ($1, $2, $3, $4)
	`, snapshot.ID, snapshot.IncidentID, snapshot.Version, payload)
	return err
}

func (s *Store) LatestInvestigation(ctx context.Context, incidentID string) (investigate.Record, bool, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, incident_id, snapshot_id, snapshot_version, provider, model, status, output, validation,
		       error_category, prompt_tokens, completion_tokens, real, started_at, completed_at
		FROM ai_investigations WHERE incident_id = $1 ORDER BY created_at DESC LIMIT 1
	`, incidentID)
	return scanInvestigation(row)
}

func (s *Store) BeginInvestigation(ctx context.Context, record investigate.Record) (bool, error) {
	output, err := json.Marshal(record.Output)
	if err != nil {
		return false, err
	}
	validation, err := json.Marshal(record.Validation)
	if err != nil {
		return false, err
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO ai_investigations (
			id, incident_id, snapshot_id, snapshot_version, provider, model, status, output, validation,
			error_category, prompt_tokens, completion_tokens, real, started_at, completed_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
	`, record.ID, record.IncidentID, record.SnapshotID, record.SnapshotVersion, record.Provider, record.Model, record.Status,
		output, validation, record.ErrorCategory, record.Usage.PromptTokens, record.Usage.CompletionTokens, record.Real, record.StartedAt, record.CompletedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (s *Store) FinishInvestigation(ctx context.Context, record investigate.Record) error {
	output, err := json.Marshal(record.Output)
	if err != nil {
		return err
	}
	validation, err := json.Marshal(record.Validation)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
		UPDATE ai_investigations
		SET status = $2, output = $3, validation = $4, error_category = $5,
		    prompt_tokens = $6, completion_tokens = $7, completed_at = $8, real = $9
		WHERE id = $1
	`, record.ID, record.Status, output, validation, record.ErrorCategory, record.Usage.PromptTokens, record.Usage.CompletionTokens, record.CompletedAt, record.Real)
	return err
}

func (s *Store) AppendIncidentEvent(ctx context.Context, incidentID, title, detail, kind string) error {
	now := time.Now().UTC()
	if _, err := s.pool.Exec(ctx, `
		INSERT INTO incident_events (incident_id, at, clock, title, detail, kind)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, incidentID, now, now.Format("15:04:05"), title, detail, kind); err != nil {
		return err
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO audit_events (at, actor, incident_id, action, policy_result, approval_result, execution_status, detail)
		VALUES ($1, 'ai-investigator', $2, $3, 'not-executable', 'not-required', 'not-executed', $4)
	`, now, incidentID, kind, title+": "+detail)
	return err
}

func (s *Store) MergeTraces(ctx context.Context, incidentID string, traces []model.Trace) error {
	var metadata []byte
	err := s.pool.QueryRow(ctx, `SELECT metadata FROM incidents WHERE id = $1 AND source = 'detection'`, incidentID).Scan(&metadata)
	if err != nil {
		return err
	}
	var blob storedDetection
	if len(metadata) > 0 {
		if err := json.Unmarshal(metadata, &blob); err != nil {
			return err
		}
	}
	blob.Evidence.Traces = traces
	raw, err := json.Marshal(blob)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `UPDATE incidents SET metadata = $2, updated_at = now() WHERE id = $1`, incidentID, raw)
	return err
}

func scanInvestigation(row pgx.Row) (investigate.Record, bool, error) {
	var (
		record     investigate.Record
		output     []byte
		validation []byte
		completed  *time.Time
	)
	err := row.Scan(&record.ID, &record.IncidentID, &record.SnapshotID, &record.SnapshotVersion, &record.Provider, &record.Model,
		&record.Status, &output, &validation, &record.ErrorCategory, &record.Usage.PromptTokens, &record.Usage.CompletionTokens,
		&record.Real, &record.StartedAt, &completed)
	if errors.Is(err, pgx.ErrNoRows) {
		return investigate.Record{}, false, nil
	}
	if err != nil {
		return investigate.Record{}, false, err
	}
	if len(output) > 0 {
		_ = json.Unmarshal(output, &record.Output)
	}
	if len(validation) > 0 {
		_ = json.Unmarshal(validation, &record.Validation)
	}
	record.CompletedAt = completed
	return record, true, nil
}
