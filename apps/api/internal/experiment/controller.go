package experiment

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"time"

	"github.com/opspilot/opspilot/apps/api/internal/model"
	"github.com/opspilot/opspilot/apps/api/internal/repository"
)

var experimentID = regexp.MustCompile(`^exp-[0-9a-f]{8}$`)

// LabStore persists real local experiments. It does not touch Kubernetes objects.
type LabStore interface {
	SaveExperiment(ctx context.Context, item model.Experiment, parameters map[string]string) error
	GetExperiment(ctx context.Context, id string) (model.Experiment, error)
	ListExperimentsReal(ctx context.Context) ([]model.Experiment, error)
	MarkExperiment(ctx context.Context, id, status string) error
	RunningExperiment(ctx context.Context) (bool, error)
}

// FaultSetter talks only to the payment-api fault endpoint.
type FaultSetter interface {
	SetPaymentFault(ctx context.Context, token, mode string, until time.Time) error
}

type Controller struct {
	Policy Policy
	Token  string
	Fault  FaultSetter
	Store  LabStore
	Now    func() time.Time
}

func (c *Controller) now() time.Time {
	if c.Now != nil {
		return c.Now()
	}
	return time.Now()
}

func (c *Controller) Start(ctx context.Context, serviceID, scenario string, durationSec int) (model.Experiment, error) {
	if err := c.Policy.Validate(serviceID, scenario, durationSec); err != nil {
		if !c.Policy.Allow {
			return model.Experiment{}, fmt.Errorf("%w: %s", repository.ErrForbidden, err.Error())
		}
		return model.Experiment{}, fmt.Errorf("%w: %s", repository.ErrInvalid, err.Error())
	}
	if c.Fault == nil || c.Store == nil || c.Token == "" {
		return model.Experiment{}, fmt.Errorf("%w: real experiments are not configured", repository.ErrForbidden)
	}
	running, err := c.Store.RunningExperiment(ctx)
	if err != nil {
		return model.Experiment{}, err
	}
	if running {
		return model.Experiment{}, fmt.Errorf("%w: a payment-api experiment is already running", repository.ErrConflict)
	}
	start := c.now()
	until := start.Add(time.Duration(durationSec) * time.Second)
	item := model.Experiment{
		ID:           "exp-" + randomID(),
		ServiceID:    ServiceID,
		ServiceName:  Service,
		Scenario:     ScenarioPaymentDegraded,
		ScenarioName: "Elevated Latency + Errors",
		DurationSec:  durationSec,
		Status:       "running",
		Simulated:    false,
		StartedAt:    start,
		EndsAt:       until,
		Note:         "Local demo-shop only. Experiment automatically expires.",
	}
	if err := c.Store.SaveExperiment(ctx, item, Parameters()); err != nil {
		return model.Experiment{}, err
	}
	if err := c.Fault.SetPaymentFault(ctx, c.Token, "degraded", until); err != nil {
		_ = c.Store.MarkExperiment(ctx, item.ID, "failed")
		return model.Experiment{}, fmt.Errorf("%w: fault injection failed: %v", repository.ErrInvalid, err)
	}
	return item, nil
}

func (c *Controller) Stop(ctx context.Context, id string) (model.Experiment, error) {
	if !experimentID.MatchString(id) {
		return model.Experiment{}, fmt.Errorf("%w: experiment id is not allowed", repository.ErrInvalid)
	}
	if !c.Policy.Allow {
		return model.Experiment{}, fmt.Errorf("%w: real experiments are disabled outside local development", repository.ErrForbidden)
	}
	if c.Fault == nil || c.Store == nil || c.Token == "" {
		return model.Experiment{}, fmt.Errorf("%w: real experiments are not configured", repository.ErrForbidden)
	}
	item, err := c.Store.GetExperiment(ctx, id)
	if err != nil {
		return model.Experiment{}, err
	}
	if item.Status != "running" {
		return item, nil
	}
	if err := c.Fault.SetPaymentFault(ctx, c.Token, "off", time.Time{}); err != nil {
		return model.Experiment{}, fmt.Errorf("%w: fault clear failed", repository.ErrInvalid)
	}
	if err := c.Store.MarkExperiment(ctx, id, "stopped"); err != nil {
		return model.Experiment{}, err
	}
	item.Status = "stopped"
	item.Note = "Fault injection cleared. This is not a Kubernetes rollback."
	return item, nil
}

func (c *Controller) Expire(ctx context.Context) error {
	if c.Store == nil {
		return nil
	}
	runs, err := c.Store.ListExperimentsReal(ctx)
	if err != nil {
		return err
	}
	now := c.now()
	for _, run := range runs {
		if run.Status == "running" && !now.Before(run.EndsAt) {
			if _, err := c.Stop(ctx, run.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

func randomID() string {
	buf := make([]byte, 4)
	if _, err := rand.Read(buf); err != nil {
		return "00000000"
	}
	return hex.EncodeToString(buf)
}
