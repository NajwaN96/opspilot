package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/opspilot/opspilot/apps/api/internal/model"
	"github.com/opspilot/opspilot/apps/api/internal/release"
	"github.com/opspilot/opspilot/apps/api/internal/repository"
)

func (s *Store) BeginPaymentRollback(ctx context.Context, incidentID string) (model.Remediation, bool, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return model.Remediation{}, false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var status string
	err = tx.QueryRow(ctx, `
		SELECT status FROM incidents WHERE id = $1 AND source = 'detection' FOR UPDATE
	`, incidentID).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Remediation{}, false, fmt.Errorf("%w: incident %s", repository.ErrNotFound, incidentID)
	}
	if err != nil {
		return model.Remediation{}, false, err
	}
	current, err := loadPaymentRemediation(ctx, tx, incidentID)
	if err != nil {
		return model.Remediation{}, false, err
	}
	if status == "resolved" || (current != nil && (current.Status == "rolling" || current.Status == "verifying" || current.Status == "succeeded")) {
		if current == nil {
			return model.Remediation{}, false, fmt.Errorf("%w: incident is resolved", repository.ErrInvalid)
		}
		return *current, false, nil
	}

	now := time.Now().UTC()
	proposal := paymentProposalID(incidentID)
	execution := paymentExecutionID(incidentID)
	if _, err := tx.Exec(ctx, `
		INSERT INTO remediation_proposals (
			id, incident_id, action, from_version, to_version, risk, policy_result, status, simulated, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, 'allowed', 'approved', false, $7)
		ON CONFLICT (id) DO UPDATE SET
			status = 'approved', policy_result = 'allowed', simulated = false, updated_at = $7
	`, proposal, incidentID, release.Action, release.BadVersion, release.GoodVersion,
		"Only demo-shop/payment-api moves from 1.5.0-bad to 1.4.2.", now); err != nil {
		return model.Remediation{}, false, err
	}
	approval := fmt.Sprintf("apr-%s-%d", strings.TrimPrefix(incidentID, "INC-REAL-"), now.UnixNano())
	if _, err := tx.Exec(ctx, `
		INSERT INTO approvals (id, proposal_id, incident_id, actor, decision)
		VALUES ($1, $2, $3, $4, 'approved')
	`, approval, proposal, incidentID, demoActor); err != nil {
		return model.Remediation{}, false, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO remediation_executions (
			id, proposal_id, incident_id, status, simulated, started_at, verification_result, updated_at
		) VALUES ($1, $2, $3, 'rolling', false, $4, '', $4)
		ON CONFLICT (id) DO UPDATE SET
			status = 'rolling', simulated = false, started_at = $4, finished_at = NULL,
			verification_result = '', updated_at = $4
	`, execution, proposal, incidentID, now); err != nil {
		return model.Remediation{}, false, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE incidents SET status = 'mitigating', updated_at = $2 WHERE id = $1 AND status <> 'resolved'
	`, incidentID, now); err != nil {
		return model.Remediation{}, false, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events (at, actor, incident_id, proposal_id, action, policy_result, approval_result, execution_status, verification_result, detail)
		VALUES ($1, $2, $3, $4, 'approval', 'allowed', 'approved', 'rolling', 'pending', $5)
	`, now, demoActor, incidentID, proposal, "Human approval recorded for rollback-payment-api before the Deployment update."); err != nil {
		return model.Remediation{}, false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return model.Remediation{}, false, err
	}
	view, err := s.paymentRemediation(ctx, incidentID)
	if err != nil {
		return model.Remediation{}, false, err
	}
	if view == nil {
		return model.Remediation{}, false, fmt.Errorf("payment rollback was not recorded")
	}
	return *view, true, nil
}

func (s *Store) MarkPaymentExecution(ctx context.Context, incidentID, status, detail string) error {
	now := time.Now().UTC()
	tag, err := s.pool.Exec(ctx, `
		UPDATE remediation_executions
		SET status = $2, updated_at = $3
		WHERE incident_id = $1 AND simulated = false
	`, incidentID, status, now)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: payment execution %s", repository.ErrNotFound, incidentID)
	}
	if status == "verifying" {
		if err := s.setDetectedVersion(ctx, incidentID, release.GoodVersion); err != nil {
			return err
		}
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO audit_events (at, actor, incident_id, proposal_id, action, policy_result, approval_result, execution_status, verification_result, detail)
		VALUES ($1, $2, $3, $4, 'rollout', 'allowed', 'approved', $5, 'pending', $6)
	`, now, "payment-rollback", incidentID, paymentProposalID(incidentID), status, detail)
	return err
}

func (s *Store) CompletePaymentVerification(ctx context.Context, incidentID, result, detail string, resolve bool) error {
	now := time.Now().UTC()
	status := "failed"
	if resolve && result == "healthy" {
		status = "succeeded"
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `
		UPDATE remediation_executions
		SET status = $2, verification_result = $3, finished_at = $4, updated_at = $4
		WHERE incident_id = $1 AND simulated = false AND status = 'verifying'
	`, incidentID, status, result, now)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return nil
	}
	if status == "succeeded" {
		if _, err := tx.Exec(ctx, `
			UPDATE incidents SET status = 'resolved', resolved_at = $2, updated_at = $2
			WHERE id = $1 AND status <> 'resolved'
		`, incidentID, now); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO incident_events (incident_id, at, clock, title, detail, kind)
			VALUES ($1, $2, $3, 'Incident resolved', $4, 'verification')
		`, incidentID, now, now.Format("15:04:05"), detail); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events (at, actor, incident_id, proposal_id, action, policy_result, approval_result, execution_status, verification_result, detail)
		VALUES ($1, 'payment-verifier', $2, $3, 'verification', 'allowed', 'approved', $4, $5, $6)
	`, now, incidentID, paymentProposalID(incidentID), status, result, detail); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) VerifyingPayments(ctx context.Context) ([]string, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT incident_id FROM remediation_executions WHERE simulated = false AND status = 'verifying' ORDER BY started_at
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
	return ids, rows.Err()
}

func (s *Store) NoteVersion(ctx context.Context, id, version string) error {
	if version == "" {
		return nil
	}
	var metadata []byte
	err := s.pool.QueryRow(ctx, `SELECT metadata FROM incidents WHERE id = $1 AND source = 'detection' AND status <> 'resolved'`, id).Scan(&metadata)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	var blob storedDetection
	if len(metadata) > 0 {
		if err := json.Unmarshal(metadata, &blob); err != nil {
			return err
		}
	}
	if blob.Facts.Version == version {
		return nil
	}
	blob.Facts.Version = version
	blob.Snapshot.Version = version
	raw, err := json.Marshal(blob)
	if err != nil {
		return err
	}
	if _, err := s.pool.Exec(ctx, `UPDATE incidents SET metadata = $2, updated_at = now() WHERE id = $1`, id, raw); err != nil {
		return err
	}
	if version != release.BadVersion {
		return nil
	}
	var exists bool
	if err := s.pool.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM incident_events WHERE incident_id = $1 AND title = 'Known bad release observed')
	`, id).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return nil
	}
	now := time.Now().UTC()
	_, err = s.pool.Exec(ctx, `
		INSERT INTO incident_events (incident_id, at, clock, title, detail, kind)
		VALUES ($1, $2, $3, 'Known bad release observed', $4, 'kubernetes')
	`, id, now, now.Format("15:04:05"), "payment-api version is 1.5.0-bad. Rollback to 1.4.2 can be approved.")
	return err
}

func (s *Store) setDetectedVersion(ctx context.Context, id, version string) error {
	return s.NoteVersion(ctx, id, version)
}

func (s *Store) paymentRemediation(ctx context.Context, incidentID string) (*model.Remediation, error) {
	return loadPaymentRemediation(ctx, s.pool, incidentID)
}

type queryRower interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func loadPaymentRemediation(ctx context.Context, q queryRower, incidentID string) (*model.Remediation, error) {
	var (
		id, action, from, to, status, verification string
		started                                    time.Time
		simulated                                  bool
	)
	err := q.QueryRow(ctx, `
		SELECT e.id, p.action, p.from_version, p.to_version, e.status, e.simulated, e.started_at, e.verification_result
		FROM remediation_executions e
		JOIN remediation_proposals p ON p.id = e.proposal_id
		WHERE e.incident_id = $1 AND e.simulated = false
	`, incidentID).Scan(&id, &action, &from, &to, &status, &simulated, &started, &verification)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &model.Remediation{
		ID: id, Action: action, From: from, To: to, Status: status, Simulated: simulated, StartedAt: started,
		Steps: paymentSteps(status, verification, started),
	}, nil
}

func paymentSteps(status, verification string, started time.Time) []model.RemediationStep {
	approval := model.RemediationStep{Name: "Approval", Status: "complete", At: &started, Clock: started.UTC().Format("15:04:05")}
	rollout := model.RemediationStep{Name: "Rollout", Status: "pending"}
	check := model.RemediationStep{Name: "Verification", Status: "pending"}
	switch status {
	case "rolling":
		rollout.Status = "active"
	case "verifying":
		rollout.Status = "complete"
		rollout.At = &started
		check.Status = "active"
	case "succeeded":
		rollout.Status = "complete"
		rollout.At = &started
		check.Status = "complete"
		check.At = &started
	case "failed":
		rollout.Status = "complete"
		rollout.At = &started
		if verification == "failed" || verification == "unhealthy" {
			check.Status = "failed"
		} else {
			rollout.Status = "failed"
		}
	}
	return []model.RemediationStep{approval, rollout, check}
}

func paymentProposalID(incidentID string) string {
	return "prop-" + strings.TrimPrefix(incidentID, "INC-REAL-")
}

func paymentExecutionID(incidentID string) string {
	return "exe-" + strings.TrimPrefix(incidentID, "INC-REAL-")
}
