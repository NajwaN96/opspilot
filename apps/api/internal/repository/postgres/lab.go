package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/opspilot/opspilot/apps/api/internal/experiment"
	"github.com/opspilot/opspilot/apps/api/internal/model"
	"github.com/opspilot/opspilot/apps/api/internal/repository"
)

func (s *Store) SaveExperiment(ctx context.Context, item model.Experiment, parameters map[string]string) error {
	raw, err := json.Marshal(parameters)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO lab_experiments (
			id, actor, service_id, service_name, scenario, scenario_name, cluster_name, namespace,
			status, simulated, duration_sec, parameters, note, started_at, expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, false, $10, $11, $12, $13, $14)
	`, item.ID, experiment.Actor, item.ServiceID, item.ServiceName, item.Scenario, item.ScenarioName,
		experiment.Cluster, experiment.Namespace, item.Status, item.DurationSec, raw, item.Note, item.StartedAt, item.EndsAt)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO audit_events (at, actor, action, policy_result, approval_result, execution_status, detail)
		VALUES ($1, $2, 'lab-fault-started', 'allowlist', 'local-operator', 'fault-enabled', $3)
	`, item.StartedAt, experiment.Actor, item.ID+" "+item.Scenario)
	return err
}

func (s *Store) GetExperiment(ctx context.Context, id string) (model.Experiment, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+experimentColumns+` FROM lab_experiments WHERE id = $1`, id)
	if err != nil {
		return model.Experiment{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		return model.Experiment{}, repository.ErrNotFound
	}
	return scanExperiment(rows)
}

func (s *Store) ListExperimentsReal(ctx context.Context) ([]model.Experiment, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+experimentColumns+` FROM lab_experiments ORDER BY started_at DESC LIMIT 20`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []model.Experiment
	for rows.Next() {
		item, err := scanExperiment(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) MarkExperiment(ctx context.Context, id, status string) error {
	now := time.Now().UTC()
	note := "Fault injection cleared. This is not a Kubernetes rollback."
	if status == "failed" {
		note = "Fault injection did not start. No Deployment was changed."
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE lab_experiments SET status = $2, stopped_at = $3, note = $4 WHERE id = $1
	`, id, status, now, note)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return repository.ErrNotFound
	}
	action := "lab-fault-cleared"
	if status == "failed" {
		action = "lab-fault-failed"
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO audit_events (at, actor, action, policy_result, approval_result, execution_status, detail)
		VALUES ($1, $2, $3, 'allowlist', 'local-operator', $4, $5)
	`, now, experiment.Actor, action, status, id)
	return err
}

func (s *Store) RunningExperiment(ctx context.Context) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM lab_experiments WHERE status = 'running')`).Scan(&exists)
	return exists, err
}

const experimentColumns = `id, service_id, service_name, scenario, scenario_name, duration_sec, status, started_at, expires_at, note`

type experimentScanner interface {
	Scan(dest ...any) error
}

func scanExperiment(row experimentScanner) (model.Experiment, error) {
	var item model.Experiment
	err := row.Scan(&item.ID, &item.ServiceID, &item.ServiceName, &item.Scenario, &item.ScenarioName, &item.DurationSec, &item.Status, &item.StartedAt, &item.EndsAt, &item.Note)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Experiment{}, repository.ErrNotFound
	}
	item.Simulated = false
	return item, err
}
