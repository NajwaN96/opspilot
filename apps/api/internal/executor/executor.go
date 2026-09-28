package executor

import (
	"context"
	"log/slog"
)

// Request is the only action shape the executor accepts.
// Callers must authorize the request before Execute is invoked.
type Request struct {
	IncidentID string
	Action     string
	Service    string
	Namespace  string
	From       string
	To         string
}

// Executor performs a constrained operational action.
// A future Kubernetes implementation must wrap a narrow client
// (for example, a rollback of one Deployment) and must not expose
// unrestricted cluster administration.
type Executor interface {
	Execute(ctx context.Context, req Request) error
}

// Simulated records the action and does not talk to a cluster.
type Simulated struct {
	Logger *slog.Logger
}

func (s Simulated) Execute(_ context.Context, req Request) error {
	logger := s.Logger
	if logger == nil {
		logger = slog.Default()
	}
	logger.Info("simulated executor accepted action",
		"incident", req.IncidentID,
		"action", req.Action,
		"service", req.Service,
		"namespace", req.Namespace,
		"from", req.From,
		"to", req.To,
	)
	return nil
}
