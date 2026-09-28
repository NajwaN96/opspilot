package repository

import (
	"context"
	"errors"

	"github.com/opspilot/opspilot/apps/api/internal/model"
)

var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("conflict")
	ErrInvalid  = errors.New("invalid")
)

// Catalog is the read/write boundary for operational state.
// The in-memory implementation can be replaced by PostgreSQL and a
// Kubernetes-backed collector without changing HTTP handlers.
type Catalog interface {
	ListClusters(ctx context.Context) ([]model.Cluster, error)
	ListServices(ctx context.Context) ([]model.Service, error)
	GetService(ctx context.Context, id string) (model.Service, error)
	ListIncidents(ctx context.Context) ([]model.Incident, error)
	GetIncident(ctx context.Context, id string) (model.Incident, error)
	StartRemediation(ctx context.Context, incidentID string) (model.Remediation, error)
	ListExperiments(ctx context.Context) (model.ExperimentCatalog, error)
	StartExperiment(ctx context.Context, serviceID, scenario string, durationSec int) (model.Experiment, error)
}
