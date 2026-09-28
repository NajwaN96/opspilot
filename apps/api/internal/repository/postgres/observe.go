package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/opspilot/opspilot/apps/api/internal/model"
)

// ApplyObservations stores a read-only discovery pass. A disconnected pass
// updates cluster status and leaves the last successful inventory in place.
func (s *Store) ApplyObservations(ctx context.Context, status model.KubernetesStatus, workloads []model.Workload, pods []model.Pod, events []model.ClusterEvent) error {
	clusterID := status.Cluster
	if clusterID == "" {
		clusterID = s.cluster
	}
	connected := status.Connectivity == "connected" || status.Connectivity == "degraded"
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `
		INSERT INTO clusters (id, name, environment, region, kubernetes_version, status, provider, simulated, source, metadata, updated_at)
		VALUES ($1, $1, 'local', 'local', $2, $3, 'k3d', false, 'kubernetes', $4, now())
		ON CONFLICT (id) DO UPDATE SET
			kubernetes_version = EXCLUDED.kubernetes_version,
			status = EXCLUDED.status,
			simulated = false,
			source = 'kubernetes',
			metadata = EXCLUDED.metadata,
			updated_at = now()
	`, clusterID, status.KubernetesVersion, status.Connectivity, metadataJSON(map[string]string{
		"mode":    status.Mode,
		"message": status.Message,
	})); err != nil {
		return err
	}
	if !connected || workloads == nil {
		return tx.Commit(ctx)
	}
	ids := make([]string, 0, len(workloads)+len(pods))
	serviceIDs := make([]string, 0, len(workloads))
	for _, workload := range workloads {
		id := resourceID(clusterID, workload.Namespace, workload.Kind, workload.Name)
		ids = append(ids, id)
		serviceIDs = append(serviceIDs, workload.ServiceID)
		labels, err := json.Marshal(workload.Labels)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO kubernetes_resources (
				id, cluster_id, namespace, kind, name, version, image, desired_replicas, ready_replicas, restarts, status, labels, last_observed_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
			ON CONFLICT (id) DO UPDATE SET
				version = EXCLUDED.version,
				image = EXCLUDED.image,
				desired_replicas = EXCLUDED.desired_replicas,
				ready_replicas = EXCLUDED.ready_replicas,
				restarts = EXCLUDED.restarts,
				status = EXCLUDED.status,
				labels = EXCLUDED.labels,
				last_observed_at = EXCLUDED.last_observed_at
		`, id, clusterID, workload.Namespace, workload.Kind, workload.Name, workload.Version, workload.Image,
			workload.Desired, workload.Ready, workload.Restarts, workload.Status, labels, workload.LastObserved); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO services (
				id, cluster_id, name, namespace, status, owner, runtime, description, version, source, telemetry,
				image, workload_kind, desired_replicas, ready_replicas, restarts, last_observed_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5, 'demo-shop', 'kubernetes',
				'Discovered from a Kubernetes Deployment. Reliability telemetry is not collected for this workload.',
				$6, 'kubernetes', 'none', $7, $8, $9, $10, $11, $12, now()
			)
			ON CONFLICT (id) DO UPDATE SET
				cluster_id = EXCLUDED.cluster_id,
				status = EXCLUDED.status,
				version = EXCLUDED.version,
				image = EXCLUDED.image,
				workload_kind = EXCLUDED.workload_kind,
				desired_replicas = EXCLUDED.desired_replicas,
				ready_replicas = EXCLUDED.ready_replicas,
				restarts = EXCLUDED.restarts,
				last_observed_at = EXCLUDED.last_observed_at,
				source = 'kubernetes',
				telemetry = 'none',
				updated_at = now()
		`, workload.ServiceID, clusterID, workload.Name, workload.Namespace, workload.Status, workload.Version,
			workload.Image, workload.Kind, workload.Desired, workload.Ready, workload.Restarts, workload.LastObserved); err != nil {
			return err
		}
	}
	for _, pod := range pods {
		id := resourceID(clusterID, pod.Namespace, "Pod", pod.Name)
		ids = append(ids, id)
		meta, err := json.Marshal(map[string]string{"ready": pod.Ready, "node": pod.Node, "service": pod.Service})
		if err != nil {
			return err
		}
		observed := time.Now().UTC()
		if pod.StartedAt != nil {
			observed = *pod.StartedAt
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO kubernetes_resources (
				id, cluster_id, namespace, kind, name, restarts, status, metadata, last_observed_at
			) VALUES ($1, $2, $3, 'Pod', $4, $5, $6, $7, $8)
			ON CONFLICT (id) DO UPDATE SET
				restarts = EXCLUDED.restarts,
				status = EXCLUDED.status,
				metadata = EXCLUDED.metadata,
				last_observed_at = EXCLUDED.last_observed_at
		`, id, clusterID, pod.Namespace, pod.Name, pod.Restarts, pod.Status, meta, observed); err != nil {
			return err
		}
	}
	namespaces := status.Namespaces
	if len(namespaces) == 0 && status.Namespace != "" {
		namespaces = []string{status.Namespace}
	}
	if len(ids) == 0 {
		ids = []string{""}
	}
	if len(serviceIDs) == 0 {
		serviceIDs = []string{""}
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM kubernetes_resources
		WHERE cluster_id = $1 AND namespace = ANY($2::text[]) AND NOT (id = ANY($3::text[]))
	`, clusterID, namespaces, ids); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM services
		WHERE source = 'kubernetes' AND cluster_id = $1 AND namespace = ANY($2::text[]) AND NOT (id = ANY($3::text[]))
	`, clusterID, namespaces, serviceIDs); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM kubernetes_events WHERE namespace = ANY($1::text[])`, namespaces); err != nil {
		return err
	}
	for _, event := range events {
		if _, err := tx.Exec(ctx, `
			INSERT INTO kubernetes_events (id, namespace, event_type, reason, object_ref, message, count, at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (id) DO UPDATE SET
				event_type = EXCLUDED.event_type,
				reason = EXCLUDED.reason,
				message = EXCLUDED.message,
				count = EXCLUDED.count,
				at = EXCLUDED.at
		`, event.ID, event.Namespace, event.Type, event.Reason, event.Object, event.Message, event.Count, event.At); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func resourceID(cluster, namespace, kind, name string) string {
	return fmt.Sprintf("%s/%s/%s/%s", cluster, namespace, kind, name)
}

func metadataJSON(value map[string]string) []byte {
	body, err := json.Marshal(value)
	if err != nil {
		return []byte("{}")
	}
	return body
}

type serviceRows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}

func scanKubernetesServices(rows serviceRows) ([]model.Service, error) {
	var out []model.Service
	for rows.Next() {
		var item model.Service
		var observed *time.Time
		if err := rows.Scan(
			&item.ID, &item.ClusterID, &item.Name, &item.Namespace, &item.Status, &item.Owner, &item.Runtime,
			&item.Description, &item.Version, &item.Image, &item.WorkloadKind, &item.Replicas.Desired, &item.Replicas.Ready,
			&item.Restarts, &observed,
		); err != nil {
			return nil, err
		}
		item.Source = "kubernetes"
		item.Telemetry = "none"
		item.LastObserved = observed
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// Ensure the scanner accepts pgx rows.
var _ serviceRows = pgx.Rows(nil)
