package postgres

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/opspilot/opspilot/apps/api/internal/rollout"
)

func (s *Store) Create(ctx context.Context, item rollout.Rollout) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO rollouts (
			id, service, cluster, namespace, stable_version, candidate_version, candidate_image,
			state, weight, stage_started_at, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
	`, item.ID, item.Service, item.Cluster, item.Namespace, item.StableVersion, item.CandidateVersion, item.CandidateImage,
		item.State, item.Weight, item.StageStarted, item.CreatedAt, item.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return errors.New("a payment-api rollout is already active")
		}
	}
	return err
}

func (s *Store) Active(ctx context.Context) (rollout.Rollout, bool, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT `+rolloutColumns+`
		FROM rollouts
		WHERE state IN ('PENDING', 'RUNNING', 'ANALYZING', 'AWAITING_APPROVAL', 'PROMOTING', 'ROLLING_BACK')
		ORDER BY created_at DESC LIMIT 1
	`)
	item, err := scanRollout(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return rollout.Rollout{}, false, nil
	}
	if err != nil {
		return rollout.Rollout{}, false, err
	}
	return item, true, nil
}

func (s *Store) Get(ctx context.Context, id string) (rollout.Rollout, error) {
	row := s.pool.QueryRow(ctx, `SELECT `+rolloutColumns+` FROM rollouts WHERE id = $1`, id)
	item, err := scanRollout(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return rollout.Rollout{}, err
	}
	return item, err
}

func (s *Store) List(ctx context.Context) ([]rollout.Rollout, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+rolloutColumns+` FROM rollouts ORDER BY created_at DESC LIMIT 20`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []rollout.Rollout
	for rows.Next() {
		item, err := scanRollout(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) Save(ctx context.Context, item rollout.Rollout) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE rollouts SET
			state = $2, weight = $3, analysis = $4, proposal_action = $5,
			ai_action = $6, ai_summary = $7, ai_status = $8, ai_mismatch = $9,
			verification = $10, requests = $11, error_rate = $12, p95 = $13,
			stable_error_rate = $14, stable_p95 = $15, candidate_ready = $16,
			stage_started_at = $17, updated_at = $18
		WHERE id = $1
	`, item.ID, item.State, item.Weight, item.Analysis, item.ProposalAction,
		item.AIAction, item.AISummary, item.AIStatus, item.AIMismatch,
		item.Verification, item.Requests, item.ErrorRate, item.P95,
		item.StableErrorRate, item.StableP95, item.CandidateReady,
		item.StageStarted, item.UpdatedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s *Store) RecordAnalysis(ctx context.Context, id string, weight int, gate rollout.Analysis) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO rollout_analysis (id, rollout_id, weight, result, requests, error_rate, p95, ready)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
	`, newRowID("ra-"), id, weight, gate.Result, gate.Requests, gate.Error, gate.P95, gate.Ready)
	return err
}

func (s *Store) AddEvent(ctx context.Context, id string, event rollout.Event) error {
	if event.At.IsZero() {
		event.At = time.Now().UTC()
	}
	if _, err := s.pool.Exec(ctx, `
		INSERT INTO rollout_events (id, rollout_id, at, title, detail, kind)
		VALUES ($1,$2,$3,$4,$5,$6)
	`, newRowID("re-"), id, event.At, event.Title, event.Detail, event.Kind); err != nil {
		return err
	}
	actor, approval, execution := "canary-controller", "not-required", "not-executed"
	if event.Kind == "approval" {
		actor, approval = "local-operator", "approved"
	}
	if event.Kind == "execution" {
		actor, execution = "canary-executor", "executed"
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO audit_events (at, actor, incident_id, action, policy_result, approval_result, execution_status, detail)
		VALUES ($1, $2, '', $3, 'checked', $4, $5, $6)
	`, event.At, actor, event.Kind, approval, execution, event.Title+": "+event.Detail)
	return err
}

func (s *Store) Events(ctx context.Context, id string) ([]rollout.Event, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT at, title, detail, kind FROM rollout_events WHERE rollout_id = $1 ORDER BY at ASC
	`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []rollout.Event
	for rows.Next() {
		var event rollout.Event
		if err := rows.Scan(&event.At, &event.Title, &event.Detail, &event.Kind); err != nil {
			return nil, err
		}
		out = append(out, event)
	}
	return out, rows.Err()
}

const rolloutColumns = `id, service, cluster, namespace, stable_version, candidate_version, candidate_image,
	state, weight, analysis, proposal_action, ai_action, ai_summary, ai_status, ai_mismatch, verification,
	requests, error_rate, p95, stable_error_rate, stable_p95, candidate_ready, stage_started_at, created_at, updated_at`

func scanRollout(row pgx.Row) (rollout.Rollout, error) {
	var item rollout.Rollout
	err := row.Scan(
		&item.ID, &item.Service, &item.Cluster, &item.Namespace, &item.StableVersion, &item.CandidateVersion, &item.CandidateImage,
		&item.State, &item.Weight, &item.Analysis, &item.ProposalAction, &item.AIAction, &item.AISummary, &item.AIStatus, &item.AIMismatch, &item.Verification,
		&item.Requests, &item.ErrorRate, &item.P95, &item.StableErrorRate, &item.StableP95, &item.CandidateReady, &item.StageStarted, &item.CreatedAt, &item.UpdatedAt,
	)
	return item, err
}

func newRowID(prefix string) string {
	var buf [4]byte
	_, _ = rand.Read(buf[:])
	return prefix + hex.EncodeToString(buf[:])
}
