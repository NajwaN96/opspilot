package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/opspilot/opspilot/apps/api/internal/model"
	"github.com/opspilot/opspilot/apps/api/internal/repository"
	"github.com/opspilot/opspilot/apps/api/internal/repository/memory"
)

const (
	demoIncident = "INC-142"
	proposalID   = "prop-INC-142"
	approvalID   = "apr-INC-142"
	executionID  = "exe-INC-142"
	demoActor    = "local-operator"
)

// Store keeps the simulated incident engine in memory and records durable
// operational state in PostgreSQL. Kubernetes observations are stored beside
// the simulation and never overwrite it.
type Store struct {
	pool    *pgxpool.Pool
	mem     *memory.Store
	env     string
	cluster string
}

type Options struct {
	DatabaseURL string
	Env         string
	Step        time.Duration
	Cluster     string
	Now         func() time.Time
}

func Open(ctx context.Context, opts Options) (*Store, error) {
	pool, err := pgxpool.New(ctx, opts.DatabaseURL)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	if err := migrate(ctx, pool); err != nil {
		pool.Close()
		return nil, err
	}
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	epoch, started, err := loadOrCreateDemo(ctx, pool, now())
	if err != nil {
		pool.Close()
		return nil, err
	}
	mem := memory.New(memory.Options{Epoch: epoch, Step: opts.Step, Now: now})
	if started != nil {
		mem.RestoreRemediation(*started)
	}
	cluster := opts.Cluster
	if cluster == "" {
		cluster = "opspilot-dev"
	}
	env := opts.Env
	if env == "" {
		env = "development"
	}
	store := &Store{pool: pool, mem: mem, env: env, cluster: cluster}
	if err := store.persistSimulation(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() {
	s.pool.Close()
}

func (s *Store) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

func (s *Store) DemoResetAllowed() bool {
	return !strings.EqualFold(s.env, "production")
}

func (s *Store) ListClusters(ctx context.Context) ([]model.Cluster, error) {
	items, err := s.mem.ListClusters(ctx)
	if err != nil {
		return nil, err
	}
	extra, err := s.kubernetesClusters(ctx)
	if err != nil {
		return nil, err
	}
	return append(items, extra...), nil
}

func (s *Store) ListServices(ctx context.Context) ([]model.Service, error) {
	items, err := s.mem.ListServices(ctx)
	if err != nil {
		return nil, err
	}
	extra, err := s.kubernetesServices(ctx)
	if err != nil {
		return nil, err
	}
	return append(items, extra...), nil
}

func (s *Store) GetService(ctx context.Context, id string) (model.Service, error) {
	if strings.HasPrefix(id, "k8s_") {
		return s.kubernetesService(ctx, id)
	}
	return s.mem.GetService(ctx, id)
}

func (s *Store) ListIncidents(ctx context.Context) ([]model.Incident, error) {
	items, err := s.mem.ListIncidents(ctx)
	if err != nil {
		return nil, err
	}
	for i := range items {
		if err := s.touchIncident(ctx, items[i]); err != nil {
			return nil, err
		}
		audit, err := s.listAudit(ctx, items[i].ID)
		if err != nil {
			return nil, err
		}
		items[i].Audit = audit
	}
	return items, nil
}

func (s *Store) GetIncident(ctx context.Context, id string) (model.Incident, error) {
	item, err := s.mem.GetIncident(ctx, id)
	if err != nil {
		return model.Incident{}, err
	}
	if err := s.touchIncident(ctx, item); err != nil {
		return model.Incident{}, err
	}
	if err := s.rewriteEvents(ctx, item); err != nil {
		return model.Incident{}, err
	}
	if err := s.syncExecution(ctx, item); err != nil {
		return model.Incident{}, err
	}
	audit, err := s.listAudit(ctx, item.ID)
	if err != nil {
		return model.Incident{}, err
	}
	item.Audit = audit
	return item, nil
}

func (s *Store) StartRemediation(ctx context.Context, incidentID string) (model.Remediation, error) {
	item, err := s.mem.StartRemediation(ctx, incidentID)
	if err != nil {
		return model.Remediation{}, err
	}
	if err := s.recordApproval(ctx, incidentID, item); err != nil {
		return model.Remediation{}, err
	}
	return item, nil
}

func (s *Store) ListExperiments(ctx context.Context) (model.ExperimentCatalog, error) {
	return s.mem.ListExperiments(ctx)
}

func (s *Store) StartExperiment(ctx context.Context, serviceID, scenario string, durationSec int) (model.Experiment, error) {
	return s.mem.StartExperiment(ctx, serviceID, scenario, durationSec)
}

func (s *Store) ResetDemo(ctx context.Context) error {
	if !s.DemoResetAllowed() {
		return fmt.Errorf("%w: demo reset is disabled outside local development", repository.ErrForbidden)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	statements := []string{
		`DELETE FROM remediation_executions WHERE incident_id = $1`,
		`DELETE FROM approvals WHERE incident_id = $1`,
		`DELETE FROM remediation_proposals WHERE incident_id = $1`,
		`DELETE FROM audit_events WHERE incident_id = $1`,
	}
	for _, statement := range statements {
		if _, err := tx.Exec(ctx, statement, demoIncident); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE demo_state SET remediation_started_at = NULL WHERE id = 1`); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events (at, actor, incident_id, action, policy_result, approval_result, execution_status, detail)
		VALUES (now(), $1, $2, 'demo-reset', 'not-applicable', 'cleared', 'cleared', 'Development reset restored INC-142. Kubernetes resources were not deleted.')
	`, demoActor, demoIncident); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	s.mem.ClearRemediation()
	item, err := s.mem.GetIncident(ctx, demoIncident)
	if err != nil {
		return err
	}
	if err := s.touchIncident(ctx, item); err != nil {
		return err
	}
	return s.rewriteEvents(ctx, item)
}

func loadOrCreateDemo(ctx context.Context, pool *pgxpool.Pool, now time.Time) (time.Time, *time.Time, error) {
	var epoch time.Time
	var started *time.Time
	err := pool.QueryRow(ctx, `SELECT epoch, remediation_started_at FROM demo_state WHERE id = 1`).Scan(&epoch, &started)
	if err == nil {
		return epoch, started, nil
	}
	if _, err := pool.Exec(ctx, `INSERT INTO demo_state (id, epoch) VALUES (1, $1) ON CONFLICT (id) DO NOTHING`, now); err != nil {
		return time.Time{}, nil, err
	}
	err = pool.QueryRow(ctx, `SELECT epoch, remediation_started_at FROM demo_state WHERE id = 1`).Scan(&epoch, &started)
	return epoch, started, err
}
