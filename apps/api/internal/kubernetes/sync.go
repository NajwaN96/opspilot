package kubernetes

import (
	"context"
	"log/slog"
	"time"

	"github.com/opspilot/opspilot/apps/api/internal/model"
)

// Sink receives a discovery pass. Implementations must be idempotent.
type Sink interface {
	ApplyObservations(ctx context.Context, status model.KubernetesStatus, workloads []model.Workload, pods []model.Pod, events []model.ClusterEvent) error
}

// Run polls the read-only client until ctx is cancelled.
func Run(ctx context.Context, every time.Duration, reader Reader, sink Sink, logger *slog.Logger) {
	if every <= 0 {
		every = 15 * time.Second
	}
	if logger == nil {
		logger = slog.Default()
	}
	syncOnce(ctx, reader, sink, logger)
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			syncOnce(ctx, reader, sink, logger)
		}
	}
}

func syncOnce(ctx context.Context, reader Reader, sink Sink, logger *slog.Logger) {
	status, err := reader.Status(ctx)
	if err != nil {
		logger.Error("kubernetes status", "error", err)
		return
	}
	if status.Connectivity == "disconnected" {
		if err := sink.ApplyObservations(ctx, status, nil, nil, nil); err != nil {
			logger.Error("record kubernetes status", "error", err)
		}
		return
	}
	workloads, err := reader.Workloads(ctx)
	if err != nil {
		logger.Error("kubernetes workloads", "error", err)
		status.Connectivity = "degraded"
		status.Message = "Workload list failed"
		_ = sink.ApplyObservations(ctx, status, nil, nil, nil)
		return
	}
	pods, err := reader.Pods(ctx)
	if err != nil {
		logger.Error("kubernetes pods", "error", err)
		status.Connectivity = "degraded"
		status.Message = "Pod list failed"
		_ = sink.ApplyObservations(ctx, status, workloads, nil, nil)
		return
	}
	events, err := reader.Events(ctx)
	if err != nil {
		logger.Error("kubernetes events", "error", err)
		events = nil
		if status.Connectivity == "connected" {
			status.Connectivity = "degraded"
			status.Message = "Event list failed"
		}
	}
	if err := sink.ApplyObservations(ctx, status, workloads, pods, events); err != nil {
		logger.Error("persist kubernetes observations", "error", err)
		return
	}
	logger.Info("kubernetes sync", "cluster", status.Cluster, "connectivity", status.Connectivity, "workloads", len(workloads), "pods", len(pods))
}
