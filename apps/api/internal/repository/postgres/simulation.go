package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/opspilot/opspilot/apps/api/internal/model"
	"github.com/opspilot/opspilot/apps/api/internal/repository"
)

func (s *Store) persistSimulation(ctx context.Context) error {
	clusters, err := s.mem.ListClusters(ctx)
	if err != nil {
		return err
	}
	for _, cluster := range clusters {
		if err := s.upsertSimulatedCluster(ctx, cluster); err != nil {
			return err
		}
	}
	services, err := s.mem.ListServices(ctx)
	if err != nil {
		return err
	}
	for _, svc := range services {
		if err := s.upsertSimulatedService(ctx, svc); err != nil {
			return err
		}
	}
	incidents, err := s.mem.ListIncidents(ctx)
	if err != nil {
		return err
	}
	for _, incident := range incidents {
		if err := s.touchIncident(ctx, incident); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) upsertSimulatedCluster(ctx context.Context, cluster model.Cluster) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO clusters (id, name, environment, region, kubernetes_version, status, provider, simulated, source, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, true, 'simulation', now())
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			environment = EXCLUDED.environment,
			region = EXCLUDED.region,
			kubernetes_version = EXCLUDED.kubernetes_version,
			status = EXCLUDED.status,
			provider = EXCLUDED.provider,
			simulated = true,
			source = 'simulation',
			updated_at = now()
	`, cluster.ID, cluster.Name, cluster.Environment, cluster.Region, cluster.KubernetesVersion, cluster.Status, cluster.Provider)
	return err
}

func (s *Store) upsertSimulatedService(ctx context.Context, svc model.Service) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO services (
			id, cluster_id, name, namespace, status, owner, runtime, description, version, previous_version,
			source, telemetry, desired_replicas, ready_replicas, availability, p95_latency_ms, error_rate, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			'simulation', 'simulated', $11, $12, $13, $14, $15, now()
		)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			version = EXCLUDED.version,
			previous_version = EXCLUDED.previous_version,
			desired_replicas = EXCLUDED.desired_replicas,
			ready_replicas = EXCLUDED.ready_replicas,
			availability = EXCLUDED.availability,
			p95_latency_ms = EXCLUDED.p95_latency_ms,
			error_rate = EXCLUDED.error_rate,
			source = 'simulation',
			telemetry = 'simulated',
			updated_at = now()
	`, svc.ID, svc.ClusterID, svc.Name, svc.Namespace, svc.Status, svc.Owner, svc.Runtime, svc.Description,
		svc.Version, svc.PreviousVersion, svc.Replicas.Desired, svc.Replicas.Ready, svc.Availability, svc.P95LatencyMs, svc.ErrorRate)
	return err
}

func (s *Store) touchIncident(ctx context.Context, incident model.Incident) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO incidents (id, severity, service_id, service_name, title, status, started_at, resolved_at, summary, cluster_name, source, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 'simulation', now())
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			resolved_at = EXCLUDED.resolved_at,
			summary = EXCLUDED.summary,
			updated_at = now()
	`, incident.ID, incident.Severity, incident.ServiceID, incident.ServiceName, incident.Title, incident.Status,
		incident.StartedAt, incident.ResolvedAt, incident.Summary, incident.Cluster)
	return err
}

func (s *Store) rewriteEvents(ctx context.Context, incident model.Incident) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `DELETE FROM incident_events WHERE incident_id = $1`, incident.ID); err != nil {
		return err
	}
	for _, event := range incident.Timeline {
		if _, err := tx.Exec(ctx, `
			INSERT INTO incident_events (incident_id, at, clock, title, detail, kind)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, incident.ID, event.At, event.Clock, event.Title, event.Detail, event.Kind); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *Store) recordApproval(ctx context.Context, incidentID string, remediation model.Remediation) error {
	incident, err := s.mem.GetIncident(ctx, incidentID)
	if err != nil {
		return err
	}
	if err := s.touchIncident(ctx, incident); err != nil {
		return err
	}
	if err := s.rewriteEvents(ctx, incident); err != nil {
		return err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `
		INSERT INTO remediation_proposals (id, incident_id, action, from_version, to_version, risk, policy_result, status, simulated)
		VALUES ($1, $2, $3, $4, $5, 'LOW', 'allowed', 'approved', true)
		ON CONFLICT (id) DO NOTHING
	`, proposalID, incidentID, remediation.Action, remediation.From, remediation.To)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return tx.Commit(ctx)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO approvals (id, proposal_id, incident_id, actor, decision)
		VALUES ($1, $2, $3, $4, 'approved')
	`, approvalID, proposalID, incidentID, demoActor); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO remediation_executions (id, proposal_id, incident_id, status, simulated, started_at, verification_result)
		VALUES ($1, $2, $3, $4, true, $5, 'pending')
	`, executionID, proposalID, incidentID, remediation.Status, remediation.StartedAt); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE demo_state SET remediation_started_at = $1 WHERE id = 1`, remediation.StartedAt); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events (at, actor, incident_id, proposal_id, action, policy_result, approval_result, execution_status, verification_result, detail)
		VALUES ($1, $2, $3, $4, $5, 'allowed', 'approved', $6, 'pending', 'Policy allowed the simulated rollback. The executor did not call Kubernetes.')
	`, remediation.StartedAt, demoActor, incidentID, proposalID, remediation.Action, remediation.Status); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) syncExecution(ctx context.Context, incident model.Incident) error {
	if incident.Remediation == nil {
		return nil
	}
	var finished *time.Time
	verification := "pending"
	if incident.Remediation.Status == "succeeded" {
		finished = incident.ResolvedAt
		verification = "healthy"
	}
	if _, err := s.pool.Exec(ctx, `
		UPDATE remediation_executions
		SET status = $2, verification_result = $3, finished_at = $4, updated_at = now()
		WHERE id = $1
	`, executionID, incident.Remediation.Status, verification, finished); err != nil {
		return err
	}
	if verification != "healthy" {
		return nil
	}
	var exists bool
	if err := s.pool.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM audit_events WHERE incident_id = $1 AND action = 'verification')
	`, incident.ID).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return nil
	}
	at := time.Now().UTC()
	if finished != nil {
		at = *finished
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO audit_events (at, actor, incident_id, proposal_id, action, policy_result, approval_result, execution_status, verification_result, detail)
		VALUES ($1, $2, $3, $4, 'verification', 'allowed', 'approved', 'succeeded', 'healthy', 'Simulated health verification passed. No live rollout was changed.')
	`, at, demoActor, incident.ID, proposalID)
	return err
}

func (s *Store) listAudit(ctx context.Context, incidentID string) ([]model.AuditRecord, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, at, actor, incident_id, COALESCE(proposal_id, ''), action, policy_result, approval_result, execution_status, verification_result, detail
		FROM audit_events
		WHERE incident_id = $1
		ORDER BY at ASC, id ASC
	`, incidentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.AuditRecord
	for rows.Next() {
		var item model.AuditRecord
		if err := rows.Scan(&item.ID, &item.At, &item.Actor, &item.IncidentID, &item.ProposalID, &item.Action, &item.PolicyResult, &item.ApprovalResult, &item.ExecutionStatus, &item.VerificationResult, &item.Detail); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) kubernetesClusters(ctx context.Context) ([]model.Cluster, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, environment, region, kubernetes_version, status, provider
		FROM clusters
		WHERE source = 'kubernetes'
		ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Cluster
	for rows.Next() {
		var item model.Cluster
		if err := rows.Scan(&item.ID, &item.Name, &item.Environment, &item.Region, &item.KubernetesVersion, &item.Status, &item.Provider); err != nil {
			return nil, err
		}
		item.Simulated = false
		item.Health.State = item.Status
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) kubernetesServices(ctx context.Context) ([]model.Service, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, cluster_id, name, namespace, status, owner, runtime, description, version, image, workload_kind,
		       desired_replicas, ready_replicas, restarts, last_observed_at
		FROM services
		WHERE source = 'kubernetes'
		ORDER BY namespace, name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanKubernetesServices(rows)
}

func (s *Store) kubernetesService(ctx context.Context, id string) (model.Service, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, cluster_id, name, namespace, status, owner, runtime, description, version, image, workload_kind,
		       desired_replicas, ready_replicas, restarts, last_observed_at
		FROM services
		WHERE source = 'kubernetes' AND id = $1
	`, id)
	if err != nil {
		return model.Service{}, err
	}
	defer rows.Close()
	items, err := scanKubernetesServices(rows)
	if err != nil {
		return model.Service{}, err
	}
	if len(items) == 0 {
		return model.Service{}, fmt.Errorf("%w: service %s", repository.ErrNotFound, id)
	}
	return items[0], nil
}
