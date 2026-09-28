package kubernetes

import (
	"context"

	"github.com/opspilot/opspilot/apps/api/internal/model"
)

// Disconnected reports a cluster that is configured but not reachable.
// It never invents workloads.
type Disconnected struct {
	Cluster   string
	Namespace string
	Scope     []string
	Message   string
}

func (d Disconnected) Status(context.Context) (model.KubernetesStatus, error) {
	namespaces := d.Scope
	if len(namespaces) == 0 && d.Namespace != "" {
		namespaces = []string{d.Namespace}
	}
	message := d.Message
	if message == "" {
		message = "No Kubernetes configuration is available"
	}
	cluster := d.Cluster
	if cluster == "" {
		cluster = "opspilot-dev"
	}
	return model.KubernetesStatus{
		Cluster:      cluster,
		Mode:         "unavailable",
		Namespace:    d.Namespace,
		Namespaces:   namespaces,
		Connectivity: "disconnected",
		Message:      message,
		Source:       "none",
	}, nil
}

func (d Disconnected) Namespaces(context.Context) ([]model.Namespace, error) {
	return []model.Namespace{}, nil
}

func (d Disconnected) Workloads(context.Context) ([]model.Workload, error) {
	return []model.Workload{}, nil
}

func (d Disconnected) Workload(context.Context, string, string) (model.Workload, error) {
	return model.Workload{}, ErrDisconnected
}

func (d Disconnected) Pods(context.Context) ([]model.Pod, error) {
	return []model.Pod{}, nil
}

func (d Disconnected) Events(context.Context) ([]model.ClusterEvent, error) {
	return []model.ClusterEvent{}, nil
}
