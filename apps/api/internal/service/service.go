package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/opspilot/opspilot/apps/api/internal/executor"
	"github.com/opspilot/opspilot/apps/api/internal/experiment"
	"github.com/opspilot/opspilot/apps/api/internal/model"
	"github.com/opspilot/opspilot/apps/api/internal/release"
	"github.com/opspilot/opspilot/apps/api/internal/repository"
)

// Service is the control-plane use-case layer.
// Policy runs here, before any executor is allowed to act.
type realLab interface {
	Start(ctx context.Context, serviceID, scenario string, durationSec int) (model.Experiment, error)
	Stop(ctx context.Context, id string) (model.Experiment, error)
}

type paymentLedger interface {
	BeginPaymentRollback(ctx context.Context, incidentID string) (model.Remediation, bool, error)
	MarkPaymentExecution(ctx context.Context, incidentID, status, detail string) error
}

type paymentWatcher interface {
	Watch(incidentID string)
}

type Service struct {
	repo           repository.Catalog
	exec           executor.Executor
	allowDemoReset bool
	allowRollouts  bool
	lab            realLab
	payment        executor.PaymentRollback
	releaser       executor.PaymentMutator
	watcher        paymentWatcher
	busy           func(context.Context) error
	restore        func(context.Context) error
}

func New(repo repository.Catalog, exec executor.Executor) *Service {
	return &Service{repo: repo, exec: exec}
}

// SetDemoResetEnabled arms the development-only INC-142 reset.
// Production processes leave this false.
func (s *Service) SetDemoResetEnabled(enabled bool) {
	s.allowDemoReset = enabled
}

// SetLab attaches the constrained local experiment controller.
func (s *Service) SetLab(lab realLab) {
	s.lab = lab
}

// SetRolloutsEnabled arms the development-only bad payment release.
func (s *Service) SetRolloutsEnabled(enabled bool) {
	s.allowRollouts = enabled
}

// SetPaymentRollback attaches the constrained payment-api mutator.
// The detection engine does not receive this value.
// SetRolloutGuards keep a canary and a payment rollback from racing.
func (s *Service) SetRolloutGuards(busy, restore func(context.Context) error) {
	s.busy = busy
	s.restore = restore
}

func (s *Service) SetPaymentRollback(mutator executor.PaymentMutator, watcher paymentWatcher) {
	s.releaser = mutator
	s.payment = executor.PaymentRollback{Mutator: mutator}
	s.watcher = watcher
}

func (s *Service) ListClusters(ctx context.Context) ([]model.Cluster, error) {
	return s.repo.ListClusters(ctx)
}

func (s *Service) ListServices(ctx context.Context) ([]model.Service, error) {
	return s.repo.ListServices(ctx)
}

func (s *Service) GetService(ctx context.Context, id string) (model.Service, error) {
	return s.repo.GetService(ctx, id)
}

func (s *Service) ListIncidents(ctx context.Context) ([]model.Incident, error) {
	return s.repo.ListIncidents(ctx)
}

func (s *Service) GetIncident(ctx context.Context, id string) (model.Incident, error) {
	return s.repo.GetIncident(ctx, id)
}

func (s *Service) ListExperiments(ctx context.Context) (model.ExperimentCatalog, error) {
	return s.repo.ListExperiments(ctx)
}

// StartRemediation authorizes a proposed action, then hands a narrow
// request to the executor. The executor cannot choose a different action.
func (s *Service) StartRemediation(ctx context.Context, incidentID, action string) (model.Remediation, error) {
	if strings.HasPrefix(incidentID, "INC-REAL-") {
		if action != release.Action {
			return model.Remediation{}, fmt.Errorf("%w: detected incidents cannot run a Kubernetes rollback", repository.ErrInvalid)
		}
		return s.startPaymentRollback(ctx, incidentID, action)
	}
	if action == "" {
		return model.Remediation{}, fmt.Errorf("%w: action is required", repository.ErrInvalid)
	}
	incident, err := s.repo.GetIncident(ctx, incidentID)
	if err != nil {
		return model.Remediation{}, err
	}
	if incident.Remediation != nil {
		return *incident.Remediation, nil
	}
	if incident.Recommendation == nil || !incident.Recommendation.Allowed || incident.Recommendation.Action != action {
		return model.Remediation{}, fmt.Errorf("%w: policy denied this action", repository.ErrInvalid)
	}
	svc, err := s.repo.GetService(ctx, incident.ServiceID)
	if err != nil {
		return model.Remediation{}, err
	}
	err = s.exec.Execute(ctx, executor.Request{
		IncidentID: incident.ID,
		Action:     action,
		Service:    svc.Name,
		Namespace:  svc.Namespace,
		From:       incident.Recommendation.From,
		To:         incident.Recommendation.To,
	})
	if err != nil {
		return model.Remediation{}, err
	}
	return s.repo.StartRemediation(ctx, incidentID)
}

func (s *Service) StopExperiment(ctx context.Context, id string) (model.Experiment, error) {
	if s.lab == nil {
		return model.Experiment{}, fmt.Errorf("%w: real experiments are not configured", repository.ErrForbidden)
	}
	return s.lab.Stop(ctx, id)
}

func (s *Service) StartExperiment(ctx context.Context, serviceID, scenario string, durationSec int) (model.Experiment, error) {
	if scenario == experiment.ScenarioPaymentDegraded {
		if s.lab == nil {
			return model.Experiment{}, fmt.Errorf("%w: real experiments are disabled", repository.ErrForbidden)
		}
		return s.lab.Start(ctx, serviceID, scenario, durationSec)
	}
	if serviceID == "" || scenario == "" {
		return model.Experiment{}, fmt.Errorf("%w: service and scenario are required", repository.ErrInvalid)
	}
	if _, err := s.repo.GetService(ctx, serviceID); err != nil {
		return model.Experiment{}, err
	}
	catalog, err := s.repo.ListExperiments(ctx)
	if err != nil {
		return model.Experiment{}, err
	}
	known := false
	for _, item := range catalog.Scenarios {
		if item.ID == scenario {
			known = true
			break
		}
	}
	if !known {
		return model.Experiment{}, fmt.Errorf("%w: unknown scenario", repository.ErrInvalid)
	}
	if durationSec == 0 {
		durationSec = 60
	}
	if durationSec < 15 || durationSec > 300 {
		return model.Experiment{}, fmt.Errorf("%w: duration must be between 15 and 300 seconds", repository.ErrInvalid)
	}
	return s.repo.StartExperiment(ctx, serviceID, scenario, durationSec)
}

func (s *Service) startPaymentRollback(ctx context.Context, incidentID, action string) (model.Remediation, error) {
	incident, err := s.repo.GetIncident(ctx, incidentID)
	if err != nil {
		return model.Remediation{}, err
	}
	if incident.Remediation != nil {
		switch incident.Remediation.Status {
		case "rolling", "verifying", "succeeded":
			if incident.Remediation.Status == "verifying" && s.watcher != nil {
				s.watcher.Watch(incident.ID)
			}
			return *incident.Remediation, nil
		}
	}
	if incident.Status == "resolved" {
		if incident.Remediation != nil {
			return *incident.Remediation, nil
		}
		return model.Remediation{}, fmt.Errorf("%w: incident is resolved", repository.ErrInvalid)
	}
	if incident.Recommendation == nil || !incident.Recommendation.Allowed || incident.Recommendation.Action != action {
		return model.Remediation{}, fmt.Errorf("%w: policy denied this action", repository.ErrInvalid)
	}
	if incident.Recommendation.From != release.BadVersion || incident.Recommendation.To != release.GoodVersion {
		return model.Remediation{}, fmt.Errorf("%w: policy denied this action", repository.ErrInvalid)
	}
	if s.busy != nil {
		if err := s.busy(ctx); err != nil {
			return model.Remediation{}, fmt.Errorf("%w: %s", repository.ErrConflict, err.Error())
		}
	}
	ledger, ok := s.repo.(paymentLedger)
	if !ok || s.payment.Mutator == nil {
		return model.Remediation{}, fmt.Errorf("%w: payment rollback is not configured", repository.ErrForbidden)
	}
	started, proceed, err := ledger.BeginPaymentRollback(ctx, incident.ID)
	if err != nil {
		return model.Remediation{}, err
	}
	if !proceed {
		if started.Status == "verifying" && s.watcher != nil {
			s.watcher.Watch(incident.ID)
		}
		return started, nil
	}
	err = s.payment.Execute(ctx, executor.Request{
		IncidentID: incident.ID,
		Action:     action,
		Service:    release.Deployment,
		Namespace:  release.Namespace,
		From:       release.BadVersion,
		To:         release.GoodVersion,
	})
	if err != nil {
		_ = ledger.MarkPaymentExecution(ctx, incident.ID, "failed", err.Error())
		if errors.Is(err, release.ErrDenied) {
			return model.Remediation{}, fmt.Errorf("%w: %s", repository.ErrInvalid, err.Error())
		}
		return model.Remediation{}, err
	}
	detail := "demo-shop/payment-api image is " + release.GoodImage + ". Prometheus verification is still running."
	if err := ledger.MarkPaymentExecution(ctx, incident.ID, "verifying", detail); err != nil {
		return model.Remediation{}, err
	}
	if s.watcher != nil {
		s.watcher.Watch(incident.ID)
	}
	updated, err := s.repo.GetIncident(ctx, incident.ID)
	if err != nil {
		return model.Remediation{}, err
	}
	if updated.Remediation == nil {
		return model.Remediation{}, fmt.Errorf("payment rollback was not recorded")
	}
	return *updated.Remediation, nil
}

// DeployBadPayment rolls payment-api to the known bad release. Development only.
// The request carries no image, namespace, or manifest.
func (s *Service) DeployBadPayment(ctx context.Context) error {
	if !s.allowRollouts || s.releaser == nil {
		return fmt.Errorf("%w: bad payment releases are disabled", repository.ErrForbidden)
	}
	if s.busy != nil {
		if err := s.busy(ctx); err != nil {
			return fmt.Errorf("%w: %s", repository.ErrConflict, err.Error())
		}
	}
	if err := s.releaser.SetPaymentRelease(ctx, release.BadImage, release.BadVersion); err != nil {
		if errors.Is(err, release.ErrDenied) {
			return fmt.Errorf("%w: %s", repository.ErrInvalid, err.Error())
		}
		return err
	}
	if err := s.releaser.WaitPaymentReady(ctx, release.BadImage, 90*time.Second); err != nil {
		return err
	}
	return nil
}

// ResetDemo restores only the seeded INC-142 simulation.
func (s *Service) ResetDemo(ctx context.Context) error {
	if !s.allowDemoReset {
		return fmt.Errorf("%w: demo reset is disabled outside local development", repository.ErrForbidden)
	}
	if s.restore != nil {
		if err := s.restore(ctx); err != nil {
			return err
		}
	}
	return s.repo.ResetDemo(ctx)
}
