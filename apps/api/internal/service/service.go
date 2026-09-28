package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/opspilot/opspilot/apps/api/internal/executor"
	"github.com/opspilot/opspilot/apps/api/internal/experiment"
	"github.com/opspilot/opspilot/apps/api/internal/model"
	"github.com/opspilot/opspilot/apps/api/internal/repository"
)

// Service is the control-plane use-case layer.
// Policy runs here, before any executor is allowed to act.
type realLab interface {
	Start(ctx context.Context, serviceID, scenario string, durationSec int) (model.Experiment, error)
	Stop(ctx context.Context, id string) (model.Experiment, error)
}

type Service struct {
	repo           repository.Catalog
	exec           executor.Executor
	allowDemoReset bool
	lab            realLab
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
		return model.Remediation{}, fmt.Errorf("%w: detected incidents cannot run a Kubernetes rollback", repository.ErrInvalid)
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

// ResetDemo restores only the seeded INC-142 simulation.
func (s *Service) ResetDemo(ctx context.Context) error {
	if !s.allowDemoReset {
		return fmt.Errorf("%w: demo reset is disabled outside local development", repository.ErrForbidden)
	}
	return s.repo.ResetDemo(ctx)
}
