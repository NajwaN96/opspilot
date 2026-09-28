package verify

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/opspilot/opspilot/apps/api/internal/detection"
	"github.com/opspilot/opspilot/apps/api/internal/release"
	"github.com/opspilot/opspilot/apps/api/internal/telemetry"
)

// Windows reads the payment-api Prometheus window. The browser does not supply the query.
type Windows interface {
	ServiceWindow(ctx context.Context, service, namespace, window string) (telemetry.Snapshot, error)
}

// Store records verification and, on success, resolves the incident.
type Store interface {
	CompletePaymentVerification(ctx context.Context, incidentID, result, detail string, resolve bool) error
	VerifyingPayments(ctx context.Context) ([]string, error)
}

// Watcher polls Prometheus after a rollout. It does not mutate Kubernetes.
type Watcher struct {
	Metrics  Windows
	Image    func(ctx context.Context) (string, error)
	Store    Store
	Logger   *slog.Logger
	Interval time.Duration
	Deadline time.Duration
	Need     int

	mu      sync.Mutex
	running map[string]bool
}

// Watch starts one background check for an incident. A second call is a no-op while the first is running.
func (w *Watcher) Watch(incidentID string) {
	if w == nil || w.Store == nil || incidentID == "" {
		return
	}
	w.mu.Lock()
	if w.running == nil {
		w.running = map[string]bool{}
	}
	if w.running[incidentID] {
		w.mu.Unlock()
		return
	}
	w.running[incidentID] = true
	w.mu.Unlock()
	go func() {
		defer func() {
			w.mu.Lock()
			delete(w.running, incidentID)
			w.mu.Unlock()
		}()
		w.run(context.Background(), incidentID)
	}()
}

// Resume reattaches verification for rollouts that were still in progress when the process started.
func (w *Watcher) Resume(ctx context.Context) {
	if w == nil || w.Store == nil {
		return
	}
	ids, err := w.Store.VerifyingPayments(ctx)
	if err != nil {
		if w.Logger != nil {
			w.Logger.Error("resume payment verification", "error", err)
		}
		return
	}
	for _, id := range ids {
		w.Watch(id)
	}
}

func (w *Watcher) run(ctx context.Context, incidentID string) {
	interval := w.Interval
	if interval <= 0 {
		interval = 15 * time.Second
	}
	limit := w.Deadline
	if limit <= 0 {
		limit = 3 * time.Minute
	}
	need := w.Need
	if need <= 0 {
		need = 2
	}
	deadline := time.Now().Add(limit)
	streak := 0
	last := "verification has not completed"
	for {
		ok, detail, err := w.healthy(ctx)
		if err != nil {
			last = err.Error()
			ok = false
		} else {
			last = detail
		}
		if ok {
			streak++
			if streak >= need {
				if err := w.Store.CompletePaymentVerification(ctx, incidentID, "healthy", detail, true); err != nil && w.Logger != nil {
					w.Logger.Error("record payment verification", "error", err, "incident", incidentID)
				}
				return
			}
		} else {
			streak = 0
		}
		if time.Now().After(deadline) {
			if err := w.Store.CompletePaymentVerification(ctx, incidentID, "failed", "Prometheus did not stay healthy on "+release.GoodImage+" before the deadline. "+last, false); err != nil && w.Logger != nil {
				w.Logger.Error("record failed payment verification", "error", err, "incident", incidentID)
			}
			return
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func (w *Watcher) healthy(ctx context.Context) (bool, string, error) {
	if w.Image == nil || w.Metrics == nil {
		return false, "", fmt.Errorf("verification is not configured")
	}
	image, err := w.Image(ctx)
	if err != nil {
		return false, "", err
	}
	if image != release.GoodImage {
		return false, "payment-api image is " + image, nil
	}
	snap, err := w.Metrics.ServiceWindow(ctx, detection.Service, detection.Namespace, "1m")
	if err != nil {
		return false, "", err
	}
	if !snap.Available {
		return false, "prometheus window unavailable", nil
	}
	decision := detection.Evaluate(detection.Sample{
		OK: true, Requests: snap.Requests, Errors: snap.Errors, P95: snap.P95,
	})
	detail := fmt.Sprintf("error rate %.1f%% and p95 %.0fms on %s", snap.ErrorRate*100, snap.P95*1000, image)
	if !decision.Healthy {
		return false, detail + "; " + decision.Reason, nil
	}
	return true, detail, nil
}
