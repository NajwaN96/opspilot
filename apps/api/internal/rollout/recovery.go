package rollout

import (
	"context"
	"fmt"
	"time"

	"github.com/opspilot/opspilot/apps/api/internal/detection"
	"github.com/opspilot/opspilot/apps/api/internal/release"
)

// present overlays live Kubernetes and Alertmanager state onto a stored rollout.
// The stored weight remains the historical stage.
func (e *Engine) present(ctx context.Context, item Rollout) Rollout {
	item.StageWeight = item.Weight
	item.CandidateActivity = "unknown"
	item.LiveWeightKnown = false
	desired, desiredKnown := 0, false
	if e.Cluster != nil {
		if weight, err := e.Cluster.LiveWeight(ctx); err == nil {
			item.LiveWeight = weight
			item.LiveWeightKnown = true
		}
		if ready, want, _, _, _, err := e.Cluster.CanaryStatus(ctx); err == nil {
			desired, desiredKnown = want, true
			switch {
			case want == 0:
				item.CandidateActivity = "idle"
			case ready:
				item.CandidateActivity = "ready"
			default:
				item.CandidateActivity = "unavailable"
			}
		}
		if ready, _, version, err := e.Cluster.StableReady(ctx); err == nil && version != "" {
			item.LiveStableKnown = true
			item.LiveStableReady = ready
			item.LiveStableVersion = version
		}
	}
	if e.Incidents != nil {
		if incident, ok, err := e.Incidents.Associated(ctx, item.CreatedAt); err == nil && ok {
			item.IncidentID = incident.ID
			item.IncidentStatus = incident.Status
		}
	}
	firingKnown := false
	var firing []string
	if e.Alerts != nil {
		names, err := e.Alerts.Firing(ctx)
		if err == nil {
			firing, firingKnown = names, true
		}
	}
	item.Alerts = alertViews(firing, firingKnown, item.LiveWeightKnown, item.LiveWeight, desired, desiredKnown, item.State)
	return item
}

// reconcile repairs an active canary so a restart cannot leave routing and replicas apart.
// It does not declare success.
func (e *Engine) reconcile(ctx context.Context, item Rollout) error {
	if e.Cluster == nil {
		return nil
	}
	switch item.State {
	case StatePending, StateRunning, StateAwaiting:
	default:
		return nil
	}
	ready, desired, _, image, _, err := e.Cluster.CanaryStatus(ctx)
	if err != nil {
		return e.attention(ctx, item, "candidate status is unreadable")
	}
	if desired != 1 || image != item.CandidateImage {
		if err := e.Cluster.EnsureCanary(ctx, item.CandidateImage, item.CandidateVersion); err != nil {
			return e.attention(ctx, item, err.Error())
		}
		_ = e.Store.AddEvent(ctx, item.ID, Event{At: e.now(), Title: "Reconciled candidate deployment", Detail: item.CandidateImage, Kind: "rollout"})
	}
	_ = ready
	if item.State == StatePending && item.Weight == 0 {
		return nil
	}
	live, err := e.Cluster.LiveWeight(ctx)
	if err != nil {
		return e.attention(ctx, item, "routing weight is unreadable")
	}
	if live == item.Weight {
		return nil
	}
	if err := e.Cluster.SetCanaryWeight(ctx, e.Token, item.Weight); err != nil {
		return e.attention(ctx, item, err.Error())
	}
	return e.Store.AddEvent(ctx, item.ID, Event{
		At: e.now(), Title: "Reconciled live canary weight",
		Detail: fmt.Sprintf("routing was %d; stage is %d", live, item.Weight), Kind: "rollout",
	})
}

func (e *Engine) attention(ctx context.Context, item Rollout, detail string) error {
	if e.now().Sub(item.StageStarted) < 3*time.Minute {
		return fmt.Errorf("%s", detail)
	}
	item.State = StateAttention
	item.Verification = detail
	item.UpdatedAt = e.now()
	if err := e.Store.Save(ctx, item); err != nil {
		return err
	}
	return e.Store.AddEvent(ctx, item.ID, Event{At: item.UpdatedAt, Title: "Operator attention required", Detail: detail, Kind: "rollout"})
}

// recoverIncidents resolves a payment-api detection incident only after the
// rollout is terminal, the candidate is idle, and Prometheus is healthy.
func (e *Engine) recoverIncidents(ctx context.Context) error {
	if e.Incidents == nil || e.Cluster == nil || e.Meter == nil || e.Store == nil {
		return nil
	}
	latest, ok, err := e.Store.Latest(ctx)
	if err != nil || !ok {
		return err
	}
	if latest.State != StateAborted && latest.State != StateSucceeded {
		return nil
	}
	live, err := e.Cluster.LiveWeight(ctx)
	if err != nil || live != 0 {
		return err
	}
	ready, desired, _, _, _, err := e.Cluster.CanaryStatus(ctx)
	if err != nil || desired != 0 || ready {
		return err
	}
	stableReady, image, version, err := e.Cluster.StableReady(ctx)
	if err != nil || !stableReady {
		return err
	}
	switch latest.State {
	case StateAborted:
		if image != release.GoodImage || version != release.GoodVersion {
			return nil
		}
	case StateSucceeded:
		if image != latest.CandidateImage || version != latest.CandidateVersion {
			return nil
		}
	}
	sample, err := e.Meter.ServiceWindow(ctx, detection.Service, detection.Namespace, "1m")
	if err != nil || !sample.Available {
		return err
	}
	decision := detection.Evaluate(detection.Sample{OK: true, Requests: sample.Requests, Errors: sample.Errors, P95: sample.P95})
	if !decision.Healthy {
		return nil
	}
	items, err := e.Incidents.Recoverable(ctx, latest.CreatedAt)
	if err != nil {
		return err
	}
	for _, incident := range items {
		if incident.ID == "" || incident.Status == "resolved" || !incident.Recovered {
			continue
		}
		resolved, err := e.Incidents.ResolveRecovered(ctx, incident.ID, latest.ID, latest.State)
		if err != nil {
			return err
		}
		if !resolved {
			continue
		}
		_ = e.Store.AddEvent(ctx, latest.ID, Event{
			At: e.now(), Title: "Incident resolved after verified recovery", Detail: incident.ID, Kind: "verification",
		})
	}
	return nil
}
