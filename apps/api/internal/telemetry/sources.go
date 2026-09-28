package telemetry

import "context"

// Sources is the local telemetry boundary. PromQL and trace queries are chosen by the callers.
type Sources struct {
	Prometheus     Prometheus
	Jaeger         Jaeger
	CollectorReady func(ctx context.Context) error
}

func (s *Sources) ComponentStatus(ctx context.Context) map[string]string {
	status := map[string]string{
		"prometheus":    "unavailable",
		"opentelemetry": "unavailable",
		"traces":        "unavailable",
	}
	if s == nil {
		return status
	}
	if err := s.Prometheus.Ready(ctx); err == nil {
		status["prometheus"] = "connected"
	}
	if s.CollectorReady != nil {
		if err := s.CollectorReady(ctx); err == nil {
			status["opentelemetry"] = "connected"
		}
	}
	if err := s.Jaeger.Ready(ctx); err == nil {
		status["traces"] = "connected"
	}
	return status
}
