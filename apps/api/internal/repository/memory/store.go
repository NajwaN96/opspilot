package memory

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/opspilot/opspilot/apps/api/internal/model"
	"github.com/opspilot/opspilot/apps/api/internal/repository"
)

const (
	clusterID       = "production-01"
	storyNowMinute  = 14*60 + 39
	paymentIncident = "INC-142"
)

// Options controls the in-memory simulation clock.
// Step of 0 completes an approved remediation immediately, which tests use.
type Options struct {
	Now   func() time.Time
	Epoch time.Time
	Step  time.Duration
}

type Store struct {
	mu          sync.Mutex
	now         func() time.Time
	epoch       time.Time
	step        time.Duration
	remediation *remediationRun
	experiments []storedExperiment
	nextExp     int
}

type remediationRun struct {
	started time.Time
}

type storedExperiment struct {
	id       string
	service  string
	scenario string
	duration time.Duration
	started  time.Time
}

func New(opts Options) *Store {
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	epoch := opts.Epoch
	if epoch.IsZero() {
		epoch = now()
	}
	return &Store{
		now:     now,
		epoch:   epoch,
		step:    opts.Step,
		nextExp: 1001,
	}
}

func (s *Store) ListClusters(context.Context) ([]model.Cluster, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return []model.Cluster{s.cluster(s.now())}, nil
}

func (s *Store) ListServices(context.Context) ([]model.Service, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	services := s.services(s.now(), false)
	return services, nil
}

func (s *Store) GetService(_ context.Context, id string) (model.Service, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, svc := range s.services(s.now(), true) {
		if svc.ID == id {
			return svc, nil
		}
	}
	return model.Service{}, fmt.Errorf("%w: service %s", repository.ErrNotFound, id)
}

func (s *Store) ListIncidents(context.Context) ([]model.Incident, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	incidents := s.incidents(s.now(), false)
	return incidents, nil
}

func (s *Store) GetIncident(_ context.Context, id string) (model.Incident, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, incident := range s.incidents(s.now(), true) {
		if incident.ID == id {
			return incident, nil
		}
	}
	return model.Incident{}, fmt.Errorf("%w: incident %s", repository.ErrNotFound, id)
}

func (s *Store) StartRemediation(ctx context.Context, incidentID string) (model.Remediation, error) {
	if incidentID != paymentIncident {
		return model.Remediation{}, fmt.Errorf("%w: incident has no executable remediation", repository.ErrConflict)
	}
	s.mu.Lock()
	if s.remediation == nil {
		s.remediation = &remediationRun{started: s.now()}
	}
	s.mu.Unlock()
	incident, err := s.GetIncident(ctx, incidentID)
	if err != nil {
		return model.Remediation{}, err
	}
	if incident.Remediation == nil {
		return model.Remediation{}, fmt.Errorf("%w: remediation was not recorded", repository.ErrConflict)
	}
	return *incident.Remediation, nil
}

func (s *Store) ListExperiments(context.Context) (model.ExperimentCatalog, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.catalog(s.now()), nil
}

func (s *Store) StartExperiment(_ context.Context, serviceID, scenario string, durationSec int) (model.Experiment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	if !serviceKnown(serviceID) {
		return model.Experiment{}, fmt.Errorf("%w: service %s", repository.ErrNotFound, serviceID)
	}
	if scenarioByID(scenario) == nil {
		return model.Experiment{}, fmt.Errorf("%w: unknown scenario", repository.ErrInvalid)
	}
	run := storedExperiment{
		id:       fmt.Sprintf("EXP-%d", s.nextExp),
		service:  serviceID,
		scenario: scenario,
		duration: time.Duration(durationSec) * time.Second,
		started:  now,
	}
	s.nextExp++
	s.experiments = append([]storedExperiment{run}, s.experiments...)
	return s.experimentView(run, now), nil
}

func (s *Store) clock(at time.Time) string {
	mins := storyNowMinute + int(math.Round(at.Sub(s.epoch).Minutes()))
	mins = ((mins % (24 * 60)) + (24 * 60)) % (24 * 60)
	return fmt.Sprintf("%02d:%02d", mins/60, mins%60)
}

func (s *Store) completedSteps(now time.Time) int {
	if s.remediation == nil {
		return 0
	}
	if s.step <= 0 {
		return 5
	}
	elapsed := now.Sub(s.remediation.started)
	if elapsed < 0 {
		elapsed = 0
	}
	n := 1 + int(elapsed/s.step)
	if n > 5 {
		return 5
	}
	return n
}

func (s *Store) cluster(now time.Time) model.Cluster {
	services := s.services(now, false)
	incidents := s.incidents(now, false)
	active := 0
	healthy := 0
	within := 0
	compliance := 0.0
	deploys := 0
	worst := model.StatusHealthy
	cutoff := now.Add(-24 * time.Hour)
	for _, svc := range services {
		compliance += svc.SLO.Compliance
		if svc.Status == model.StatusHealthy {
			healthy++
		}
		if svc.SLO.WithinSLO {
			within++
		}
		if rank(svc.Status) > rank(worst) {
			worst = svc.Status
		}
		for _, dep := range s.deploymentsFor(svc.ID, now) {
			if dep.At.After(cutoff) {
				deploys++
			}
		}
	}
	for _, incident := range incidents {
		if incident.Status != model.IncidentResolved {
			active++
		}
	}
	if len(services) > 0 {
		compliance = round(compliance/float64(len(services)), 2)
	}
	return model.Cluster{
		ID:                clusterID,
		Name:              "production-01",
		Environment:       "production",
		Region:            "lab",
		KubernetesVersion: "v1.31.4",
		Status:            worst,
		Provider:          "simulated",
		Simulated:         true,
		Health: model.ClusterHealth{
			State:             worst,
			ActiveIncidents:   active,
			SLOCompliance:     compliance,
			ServicesWithinSLO: within,
			ServicesTotal:     len(services),
			ServicesHealthy:   healthy,
			DeploymentsToday:  deploys,
		},
		Nodes:        nodes(),
		Namespaces:   s.namespaces(now),
		ControlPlane: controlPlane(),
	}
}

func (s *Store) services(now time.Time, detail bool) []model.Service {
	progress := s.completedSteps(now)
	pay := paymentMetrics(progress)
	out := make([]model.Service, 0, len(serviceSpecs))
	for _, spec := range serviceSpecs {
		svc := spec.materialize(s, now, pay)
		if detail {
			svc.Dependencies = spec.dependencies(pay.status)
			svc.Deployments = s.deploymentsFor(spec.id, now)
			svc.Metrics = s.metricsFor(spec.id, now, progress)
		}
		if spec.id == "payment-api" && progress < 5 {
			svc.OpenIncidents = []string{paymentIncident}
		}
		out = append(out, svc)
	}
	return out
}

func (s *Store) incidents(now time.Time, detail bool) []model.Incident {
	items := []model.Incident{
		s.paymentIncident(now, detail),
		s.historicalCheckout(now, detail),
		s.historicalNotifications(now, detail),
	}
	return items
}

func (s *Store) paymentIncident(now time.Time, detail bool) model.Incident {
	progress := s.completedSteps(now)
	pay := paymentMetrics(progress)
	started := s.epoch.Add(-5 * time.Minute)
	status := model.IncidentInvestigating
	var resolvedAt *time.Time
	if progress > 0 && progress < 5 {
		status = model.IncidentMitigating
	}
	if progress >= 5 {
		status = model.IncidentResolved
		done := s.stepTime(4)
		resolvedAt = &done
	}
	end := now
	if resolvedAt != nil {
		end = *resolvedAt
	}
	incident := model.Incident{
		ID:          paymentIncident,
		Severity:    "SEV-2",
		ServiceID:   "payment-api",
		ServiceName: "payment-api",
		Title:       "Payment API latency and elevated error rate",
		Status:      status,
		StartedAt:   started,
		ResolvedAt:  resolvedAt,
		DurationSec: int(end.Sub(started).Seconds()),
		Summary:     "payment-api:v1.8.2 saturated the Postgres connection pool immediately after rollout. Charge requests piled up behind acquire timeouts.",
		Cluster:     clusterID,
		Snapshot: model.Snapshot{
			Availability:  pay.availability,
			P95LatencyMs:  pay.p95,
			ErrorRate:     pay.errorRate,
			DBConnections: pay.db,
			Status:        statusLabel(status),
			Version:       pay.version,
		},
		Recommendation: &model.Recommendation{
			Action:  "rollback",
			Summary: "Rollback payment-api",
			From:    "v1.8.2",
			To:      "v1.8.1",
			Risk:    "LOW",
			Policy:  "Allowed",
			Allowed: progress == 0,
		},
	}
	if s.remediation != nil {
		rem := s.remediationView(now)
		incident.Remediation = &rem
	}
	if !detail {
		return incident
	}
	incident.Timeline = s.paymentTimeline(now)
	incident.Evidence = s.paymentEvidence(now, progress)
	incident.Analysis = &model.Analysis{
		Simulated:  true,
		Cause:      "Database connection leak introduced by payment-api:v1.8.2",
		Confidence: 0.91,
		Evidence: []string{
			"Incident started immediately after deployment",
			"Database connections saturated",
			"Application logs show connection pool exhaustion",
			"Previous application version was healthy",
			"Trace latency is concentrated in database calls",
		},
	}
	return incident
}

func statusLabel(status string) string {
	switch status {
	case model.IncidentResolved:
		return "Resolved"
	case model.IncidentMitigating:
		return "Mitigating"
	default:
		return "Investigating"
	}
}

func (s *Store) paymentTimeline(now time.Time) []model.TimelineEvent {
	type seed struct {
		at     time.Time
		title  string
		detail string
		kind   string
	}
	seeds := []seed{
		{s.epoch.Add(-8 * time.Minute), "Deployment payment-api:v1.8.2", "Rolling update from v1.8.1 to v1.8.2 in namespace payments.", "deployment"},
		{s.epoch.Add(-7 * time.Minute), "P95 latency increased 280%", "payment-api p95 moved from 210ms to 1.8s.", "metric"},
		{s.epoch.Add(-7*time.Minute + 15*time.Second), "Database connections reached 94%", "postgres connection usage crossed the saturation threshold.", "metric"},
		{s.epoch.Add(-6 * time.Minute), "Error rate reached 12.4%", "POST /v1/charges returned 500s after pool acquire timeouts.", "metric"},
		{s.epoch.Add(-6*time.Minute + 20*time.Second), "SLO burn threshold exceeded", "30-day error budget burn rate crossed the page threshold.", "slo"},
		{s.epoch.Add(-5 * time.Minute), "Incident created", "INC-142 opened as SEV-2 for payment-api.", "incident"},
	}
	events := make([]model.TimelineEvent, 0, len(seeds)+5)
	for _, item := range seeds {
		events = append(events, model.TimelineEvent{
			Clock:  s.clock(item.at),
			At:     item.at,
			Title:  item.title,
			Detail: item.detail,
			Kind:   item.kind,
		})
	}
	if s.remediation == nil {
		return events
	}
	names := []struct {
		title  string
		detail string
		kind   string
	}{
		{"Approval recorded", "On-call approved the rollback proposal. Policy classified the action as LOW risk.", "remediation"},
		{"Rollback started", "Simulated executor accepted rollback of payments/payment-api from v1.8.2 to v1.8.1.", "remediation"},
		{"Deployment progressing", "Replica set payment-api is returning to the previous pod template.", "remediation"},
		{"Health verification", "Error rate, p95 latency, and postgres connections are being compared to the healthy baseline.", "remediation"},
		{"Incident resolved", "payment-api recovered on v1.8.1. No cluster admin credentials were used.", "remediation"},
	}
	n := s.completedSteps(now)
	for i := 0; i < n && i < len(names); i++ {
		at := s.stepTime(i)
		events = append(events, model.TimelineEvent{
			Clock:  s.clock(at),
			At:     at,
			Title:  names[i].title,
			Detail: names[i].detail,
			Kind:   names[i].kind,
		})
	}
	return events
}

func (s *Store) stepTime(index int) time.Time {
	if s.remediation == nil {
		return s.epoch
	}
	if s.step <= 0 {
		return s.remediation.started
	}
	return s.remediation.started.Add(time.Duration(index) * s.step)
}

func (s *Store) remediationView(now time.Time) model.Remediation {
	n := s.completedSteps(now)
	names := []string{
		"Approval recorded",
		"Rollback started",
		"Deployment progressing",
		"Health verification",
		"Incident resolved",
	}
	steps := make([]model.RemediationStep, len(names))
	for i, name := range names {
		step := model.RemediationStep{Name: name, Status: "pending"}
		if i < n {
			at := s.stepTime(i)
			step.Status = "complete"
			step.At = &at
			step.Clock = s.clock(at)
		} else if i == n {
			step.Status = "active"
		}
		steps[i] = step
	}
	status := "running"
	if n >= len(names) {
		status = "succeeded"
	}
	return model.Remediation{
		ID:        "REM-142",
		Action:    "rollback",
		From:      "v1.8.2",
		To:        "v1.8.1",
		Status:    status,
		Simulated: true,
		StartedAt: s.remediation.started,
		Steps:     steps,
	}
}

func (s *Store) paymentEvidence(now time.Time, progress int) *model.Evidence {
	return &model.Evidence{
		Metrics:          s.metricsFor("payment-api", now, progress),
		Logs:             paymentLogs(progress),
		Traces:           paymentTraces(progress),
		KubernetesEvents: paymentEvents(progress),
		Deployments:      s.deploymentsFor("payment-api", now),
	}
}

func (s *Store) historicalCheckout(now time.Time, detail bool) model.Incident {
	started := s.epoch.Add(-26 * time.Hour)
	resolved := started.Add(42 * time.Minute)
	incident := model.Incident{
		ID:          "INC-138",
		Severity:    "SEV-3",
		ServiceID:   "checkout-api",
		ServiceName: "checkout-api",
		Title:       "Checkout timeouts calling inventory",
		Status:      model.IncidentResolved,
		StartedAt:   started,
		ResolvedAt:  &resolved,
		DurationSec: int(resolved.Sub(started).Seconds()),
		Summary:     "A slow inventory query added tail latency to checkout. The query was reverted and the service recovered.",
		Cluster:     clusterID,
		Snapshot: model.Snapshot{
			Availability:  99.98,
			P95LatencyMs:  180,
			ErrorRate:     0.12,
			DBConnections: 22,
			Status:        "Resolved",
			Version:       "v2.4.1",
		},
		Recommendation: &model.Recommendation{
			Action:  "none",
			Summary: "No action. Incident already resolved.",
			Risk:    "LOW",
			Policy:  "Not required",
			Allowed: false,
		},
	}
	if !detail {
		return incident
	}
	incident.Analysis = &model.Analysis{
		Simulated:  true,
		Cause:      "Inventory availability query regressed after an index change.",
		Confidence: 0.86,
		Evidence: []string{
			"Checkout spans stalled in the inventory client",
			"inventory-api p95 recovered after the index rollback",
			"No payment or order errors were involved",
		},
	}
	incident.Timeline = []model.TimelineEvent{
		{Clock: s.clock(started), At: started, Title: "Error budget burn detected", Detail: "checkout-api availability dipped under the fast-burn threshold.", Kind: "slo"},
		{Clock: s.clock(started.Add(6 * time.Minute)), At: started.Add(6 * time.Minute), Title: "Inventory query identified", Detail: "Slow span: SELECT availability FROM stock_items.", Kind: "trace"},
		{Clock: s.clock(resolved), At: resolved, Title: "Incident resolved", Detail: "Index change reverted. Checkout latency returned to baseline.", Kind: "incident"},
	}
	incident.Evidence = &model.Evidence{
		Metrics: s.metricsFor("checkout-api", now, 0),
		Logs: []model.LogLine{
			{Clock: s.clock(started.Add(2 * time.Minute)), Level: "WARN", Service: "checkout-api", Message: "inventory-api client timeout budget exhausted"},
		},
		Traces: []model.Trace{{
			ID: "trace-138", TraceID: "4c1e138aa0", Clock: s.clock(started.Add(3 * time.Minute)),
			Spans: []model.Span{
				{ID: "s1", Service: "checkout-api", Name: "POST /v1/checkout", DurationMs: 920, Status: "ok"},
				{ID: "s2", ParentID: "s1", Service: "inventory-api", Name: "GET /v1/availability", DurationMs: 860, Status: "ok", Slow: true},
			},
		}},
		KubernetesEvents: []model.KubeEvent{
			{Clock: s.clock(resolved.Add(-2 * time.Minute)), Type: "Normal", Reason: "DeploymentRollout", Object: "deployment/inventory-api", Message: "Rolled back the availability index migration."},
		},
		Deployments: s.deploymentsFor("checkout-api", now),
	}
	return incident
}

func (s *Store) historicalNotifications(now time.Time, detail bool) model.Incident {
	started := s.epoch.Add(-72 * time.Hour)
	resolved := started.Add(28 * time.Minute)
	incident := model.Incident{
		ID:          "INC-121",
		Severity:    "SEV-3",
		ServiceID:   "notification-worker",
		ServiceName: "notification-worker",
		Title:       "Notification backlog after broker restart",
		Status:      model.IncidentResolved,
		StartedAt:   started,
		ResolvedAt:  &resolved,
		DurationSec: int(resolved.Sub(started).Seconds()),
		Summary:     "A broker restart stalled the notification consumer. The consumer caught up without message loss.",
		Cluster:     clusterID,
		Snapshot: model.Snapshot{
			Availability:  99.97,
			P95LatencyMs:  90,
			ErrorRate:     0.05,
			DBConnections: 8,
			Status:        "Resolved",
			Version:       "v1.3.4",
		},
		Recommendation: &model.Recommendation{
			Action:  "none",
			Summary: "No action. Incident already resolved.",
			Risk:    "LOW",
			Policy:  "Not required",
			Allowed: false,
		},
	}
	if !detail {
		return incident
	}
	incident.Analysis = &model.Analysis{
		Simulated:  true,
		Cause:      "Consumer group paused while the broker restarted.",
		Confidence: 0.8,
		Evidence: []string{
			"Queue lag rose immediately after the broker pod restarted",
			"Worker error rate stayed near zero",
			"Lag returned to zero after the consumer resumed",
		},
	}
	incident.Timeline = []model.TimelineEvent{
		{Clock: s.clock(started), At: started, Title: "Queue lag alert", Detail: "notification backlog passed 10,000 messages.", Kind: "metric"},
		{Clock: s.clock(resolved), At: resolved, Title: "Incident resolved", Detail: "Consumer caught up. No customer notifications were dropped.", Kind: "incident"},
	}
	incident.Evidence = &model.Evidence{
		Metrics: s.metricsFor("notification-worker", now, 0),
		Logs: []model.LogLine{
			{Clock: s.clock(started), Level: "WARN", Service: "notification-worker", Message: "consumer paused: broker coordinator unavailable"},
		},
		Traces:           []model.Trace{},
		KubernetesEvents: []model.KubeEvent{{Clock: s.clock(started.Add(-1 * time.Minute)), Type: "Normal", Reason: "Killing", Object: "pod/redis-0", Message: "Stopping container redis"}},
		Deployments:      s.deploymentsFor("notification-worker", now),
	}
	return incident
}

type payState struct {
	status       string
	version      string
	availability float64
	errorRate    float64
	db           float64
	p95          int
	compliance   float64
	budget       float64
	burn         float64
	within       bool
}

func paymentMetrics(progress int) payState {
	t := 0.0
	if progress > 0 {
		t = float64(progress-1) / 4
	}
	if progress >= 5 {
		t = 1
	}
	state := payState{
		status:       model.StatusCritical,
		version:      "v1.8.2",
		availability: round(lerp(98.71, 99.96, t), 2),
		errorRate:    round(lerp(12.4, 0.3, t), 2),
		db:           round(lerp(94, 37, t), 0),
		p95:          int(math.Round(lerp(1800, 240, t))),
		compliance:   round(lerp(98.40, 99.94, t), 2),
		budget:       round(lerp(11, 61, t), 0),
		burn:         round(lerp(18.6, 0.8, t), 1),
		within:       false,
	}
	if progress >= 5 {
		state.status = model.StatusHealthy
		state.version = "v1.8.1"
		state.within = true
	} else if t >= 0.75 {
		state.status = model.StatusDegraded
	}
	return state
}

type serviceSpec struct {
	id           string
	name         string
	namespace    string
	owner        string
	runtime      string
	description  string
	version      string
	previous     string
	availability float64
	p95          int
	errorRate    float64
	objective    float64
	compliance   float64
	budget       float64
	burn         float64
	replicas     int
	last         time.Duration
}

func (spec serviceSpec) materialize(s *Store, now time.Time, pay payState) model.Service {
	version := spec.version
	previous := spec.previous
	availability := spec.availability
	p95 := spec.p95
	errorRate := spec.errorRate
	status := model.StatusHealthy
	compliance := spec.compliance
	budget := spec.budget
	burn := spec.burn
	within := spec.compliance >= spec.objective
	last := s.epoch.Add(spec.last)
	if spec.id == "payment-api" {
		version = pay.version
		previous = "v1.8.1"
		if pay.version == "v1.8.1" {
			previous = "v1.8.2"
		}
		availability = pay.availability
		p95 = pay.p95
		errorRate = pay.errorRate
		status = pay.status
		compliance = pay.compliance
		budget = pay.budget
		burn = pay.burn
		within = pay.within
		if s.remediation != nil {
			last = s.remediation.started
		}
	}
	return model.Service{
		ID:              spec.id,
		Name:            spec.name,
		ClusterID:       clusterID,
		Namespace:       spec.namespace,
		Status:          status,
		Owner:           spec.owner,
		Runtime:         spec.runtime,
		Description:     spec.description,
		Version:         version,
		PreviousVersion: previous,
		Availability:    availability,
		P95LatencyMs:    p95,
		ErrorRate:       errorRate,
		LastDeployment:  last,
		Replicas:        model.Replicas{Desired: spec.replicas, Ready: spec.replicas},
		SLO: model.SLO{
			Objective:            spec.objective,
			Window:               "30d",
			Compliance:           compliance,
			ErrorBudgetRemaining: budget,
			BurnRate:             burn,
			WithinSLO:            within,
		},
	}
}

func (spec serviceSpec) dependencies(paymentStatus string) []model.Dependency {
	dbStatus := model.StatusHealthy
	if spec.id == "payment-api" && paymentStatus != model.StatusHealthy {
		dbStatus = model.StatusCritical
	}
	switch spec.id {
	case "checkout-api":
		return []model.Dependency{
			{ID: "payment-api", Name: "payment-api", Kind: "service", Relation: "downstream", Status: paymentStatus},
			{ID: "orders-api", Name: "orders-api", Kind: "service", Relation: "downstream", Status: model.StatusHealthy},
		}
	case "payment-api":
		return []model.Dependency{
			{ID: "checkout-api", Name: "checkout-api", Kind: "service", Relation: "upstream", Status: model.StatusHealthy},
			{ID: "orders-api", Name: "orders-api", Kind: "service", Relation: "upstream", Status: model.StatusHealthy},
			{ID: "payments-db", Name: "postgres", Kind: "datastore", Relation: "downstream", Status: dbStatus},
			{ID: "notification-worker", Name: "notification-worker", Kind: "service", Relation: "downstream", Status: model.StatusHealthy},
		}
	case "orders-api":
		return []model.Dependency{
			{ID: "checkout-api", Name: "checkout-api", Kind: "service", Relation: "upstream", Status: model.StatusHealthy},
			{ID: "inventory-api", Name: "inventory-api", Kind: "service", Relation: "downstream", Status: model.StatusHealthy},
			{ID: "payment-api", Name: "payment-api", Kind: "service", Relation: "downstream", Status: paymentStatus},
		}
	case "inventory-api":
		return []model.Dependency{
			{ID: "orders-api", Name: "orders-api", Kind: "service", Relation: "upstream", Status: model.StatusHealthy},
			{ID: "inventory-db", Name: "postgres", Kind: "datastore", Relation: "downstream", Status: model.StatusHealthy},
		}
	default:
		return []model.Dependency{
			{ID: "payment-api", Name: "payment-api", Kind: "service", Relation: "upstream", Status: paymentStatus},
			{ID: "orders-api", Name: "orders-api", Kind: "service", Relation: "upstream", Status: model.StatusHealthy},
			{ID: "notifications-redis", Name: "redis", Kind: "queue", Relation: "downstream", Status: model.StatusHealthy},
		}
	}
}

var serviceSpecs = []serviceSpec{
	{
		id: "checkout-api", name: "checkout-api", namespace: "commerce", owner: "commerce-checkout",
		runtime: "Go 1.22", description: "Authorizes carts and orchestrates payment and order creation.",
		version: "v2.4.1", previous: "v2.4.0", availability: 99.98, p95: 180, errorRate: 0.12,
		objective: 99.9, compliance: 99.98, budget: 92, burn: 0.4, replicas: 6, last: -2 * time.Hour,
	},
	{
		id: "payment-api", name: "payment-api", namespace: "payments", owner: "payments",
		runtime: "Go 1.22", description: "Charges cards and records payment intents.",
		version: "v1.8.2", previous: "v1.8.1", availability: 98.71, p95: 1800, errorRate: 12.4,
		objective: 99.9, compliance: 98.4, budget: 11, burn: 18.6, replicas: 8, last: -8 * time.Minute,
	},
	{
		id: "orders-api", name: "orders-api", namespace: "commerce", owner: "commerce-orders",
		runtime: "Go 1.22", description: "Creates and tracks customer orders.",
		version: "v3.1.0", previous: "v3.0.9", availability: 99.95, p95: 210, errorRate: 0.2,
		objective: 99.9, compliance: 99.95, budget: 88, burn: 0.6, replicas: 4, last: -5 * time.Hour,
	},
	{
		id: "inventory-api", name: "inventory-api", namespace: "supply", owner: "supply-chain",
		runtime: "Java 21", description: "Reserves stock and publishes availability.",
		version: "v4.0.6", previous: "v4.0.5", availability: 99.91, p95: 260, errorRate: 0.4,
		objective: 99.9, compliance: 99.91, budget: 74, burn: 0.9, replicas: 4, last: -20 * time.Hour,
	},
	{
		id: "notification-worker", name: "notification-worker", namespace: "platform", owner: "platform-messaging",
		runtime: "Go 1.22", description: "Delivers order and payment notifications.",
		version: "v1.3.4", previous: "v1.3.3", availability: 99.97, p95: 90, errorRate: 0.05,
		objective: 99.5, compliance: 99.97, budget: 96, burn: 0.2, replicas: 3, last: -3 * time.Hour,
	},
}

func serviceKnown(id string) bool {
	for _, spec := range serviceSpecs {
		if spec.id == id {
			return true
		}
	}
	return false
}

func (s *Store) deploymentsFor(serviceID string, now time.Time) []model.Deployment {
	progress := s.completedSteps(now)
	var items []model.Deployment
	switch serviceID {
	case "payment-api":
		badStatus := "suspect"
		if progress >= 5 {
			badStatus = "rolled-back"
		}
		items = append(items, model.Deployment{
			ID: "dep-pay-182", ServiceID: serviceID, ServiceName: serviceID,
			Version: "v1.8.2", PreviousVersion: "v1.8.1", At: s.epoch.Add(-8 * time.Minute), Clock: s.clock(s.epoch.Add(-8 * time.Minute)),
			Actor: "payments-oncall", Status: badStatus, Strategy: "RollingUpdate",
			Change: "Reuse database connections across charge retries.",
		})
		if s.remediation != nil {
			status := "in-progress"
			if progress >= 5 {
				status = "succeeded"
			}
			at := s.remediation.started
			items = append([]model.Deployment{{
				ID: "dep-pay-181-rollback", ServiceID: serviceID, ServiceName: serviceID,
				Version: "v1.8.1", PreviousVersion: "v1.8.2", At: at, Clock: s.clock(at),
				Actor: "opspilot-executor", Status: status, Strategy: "Rollback",
				Change: "Simulated rollback to the last healthy version.",
			}}, items...)
		}
		items = append(items, model.Deployment{
			ID: "dep-pay-181", ServiceID: serviceID, ServiceName: serviceID,
			Version: "v1.8.1", PreviousVersion: "v1.8.0", At: s.epoch.Add(-30 * time.Hour), Clock: s.clock(s.epoch.Add(-30 * time.Hour)),
			Actor: "payments-oncall", Status: "succeeded", Strategy: "RollingUpdate",
			Change: "Tighten charge idempotency keys.",
		})
	case "checkout-api":
		items = []model.Deployment{
			{ID: "dep-co-241", ServiceID: serviceID, ServiceName: serviceID, Version: "v2.4.1", PreviousVersion: "v2.4.0", At: s.epoch.Add(-2 * time.Hour), Clock: s.clock(s.epoch.Add(-2 * time.Hour)), Actor: "commerce-checkout", Status: "succeeded", Strategy: "RollingUpdate", Change: "Raise the inventory client timeout to match the SLO."},
			{ID: "dep-co-240", ServiceID: serviceID, ServiceName: serviceID, Version: "v2.4.0", PreviousVersion: "v2.3.4", At: s.epoch.Add(-50 * time.Hour), Clock: s.clock(s.epoch.Add(-50 * time.Hour)), Actor: "commerce-checkout", Status: "succeeded", Strategy: "RollingUpdate", Change: "Add cart revalidation before payment."},
		}
	case "orders-api":
		items = []model.Deployment{
			{ID: "dep-ord-310", ServiceID: serviceID, ServiceName: serviceID, Version: "v3.1.0", PreviousVersion: "v3.0.9", At: s.epoch.Add(-5 * time.Hour), Clock: s.clock(s.epoch.Add(-5 * time.Hour)), Actor: "commerce-orders", Status: "succeeded", Strategy: "RollingUpdate", Change: "Batch order status writes."},
		}
	case "inventory-api":
		items = []model.Deployment{
			{ID: "dep-inv-406", ServiceID: serviceID, ServiceName: serviceID, Version: "v4.0.6", PreviousVersion: "v4.0.5", At: s.epoch.Add(-20 * time.Hour), Clock: s.clock(s.epoch.Add(-20 * time.Hour)), Actor: "supply-chain", Status: "succeeded", Strategy: "RollingUpdate", Change: "Restore the availability index."},
		}
	default:
		items = []model.Deployment{
			{ID: "dep-ntf-134", ServiceID: serviceID, ServiceName: serviceID, Version: "v1.3.4", PreviousVersion: "v1.3.3", At: s.epoch.Add(-3 * time.Hour), Clock: s.clock(s.epoch.Add(-3 * time.Hour)), Actor: "platform-messaging", Status: "succeeded", Strategy: "RollingUpdate", Change: "Resume consumers after coordinator loss."},
		}
	}
	return items
}

func (s *Store) metricsFor(serviceID string, now time.Time, progress int) []model.MetricPoint {
	points := make([]model.MetricPoint, 0, 24)
	for minutesAgo := 23; minutesAgo >= 0; minutesAgo-- {
		at := s.epoch.Add(-time.Duration(minutesAgo) * time.Minute)
		p95, errRate, db, rps := baseline(serviceID)
		if serviceID == "payment-api" {
			p95, errRate, db, rps = paymentPoint(minutesAgo, progress)
		}
		points = append(points, model.MetricPoint{
			Clock:         s.clock(at),
			P95LatencyMs:  p95,
			ErrorRate:     round(errRate, 2),
			DBConnections: round(db, 0),
			RequestRate:   round(rps, 0),
		})
	}
	_ = now
	return points
}

func baseline(serviceID string) (int, float64, float64, float64) {
	switch serviceID {
	case "checkout-api":
		return 180, 0.12, 22, 310
	case "orders-api":
		return 210, 0.2, 28, 240
	case "inventory-api":
		return 260, 0.4, 31, 190
	default:
		return 90, 0.05, 8, 80
	}
}

func paymentPoint(minutesAgo, progress int) (int, float64, float64, float64) {
	p95 := 210
	errRate := 0.2
	db := 34.0
	rps := 420.0
	switch {
	case minutesAgo > 7:
		// healthy, before the bad rollout's effect
	case minutesAgo == 7:
		p95, errRate, db, rps = 780, 3.1, 71, 390
	default:
		p95, errRate, db, rps = 1800, 12.4, 94, 260
	}
	if progress > 0 && minutesAgo <= 3 {
		t := float64(progress-1) / 4
		if progress >= 5 {
			t = 1
		}
		weight := t * float64(4-minutesAgo) / 4
		if weight > 1 {
			weight = 1
		}
		p95 = int(math.Round(lerp(float64(p95), 240, weight)))
		errRate = lerp(errRate, 0.3, weight)
		db = lerp(db, 37, weight)
		rps = lerp(rps, 450, weight)
	}
	return p95, errRate, db, rps
}

func paymentLogs(progress int) []model.LogLine {
	logs := []model.LogLine{
		{Clock: "14:32", Level: "WARN", Service: "payment-api", Message: "connection acquire exceeded 250ms pool_in_use=71%"},
		{Clock: "14:32", Level: "ERROR", Service: "payment-api", Message: "database connection pool exhausted"},
		{Clock: "14:33", Level: "ERROR", Service: "payment-api", Message: "postgres: FATAL remaining connection slots are reserved"},
		{Clock: "14:33", Level: "ERROR", Service: "payment-api", Message: "POST /v1/charges status=500 duration_ms=1842 error=pool_timeout"},
		{Clock: "14:33", Level: "WARN", Service: "checkout-api", Message: "upstream payment-api deadline exceeded dependency=payment-api"},
		{Clock: "14:34", Level: "INFO", Service: "opspilot", Message: "incident INC-142 opened severity=SEV-2 service=payment-api"},
	}
	if progress >= 2 {
		logs = append(logs, model.LogLine{Clock: "14:40", Level: "INFO", Service: "payment-api", Message: "deployment rollout payment-api:v1.8.1 started"})
	}
	if progress >= 5 {
		logs = append(logs, model.LogLine{Clock: "14:42", Level: "INFO", Service: "payment-api", Message: "connection pool recovered size=32 idle=20"})
	}
	return logs
}

func paymentTraces(progress int) []model.Trace {
	dbMs := 1720
	status := "error"
	slow := true
	if progress >= 5 {
		dbMs = 18
		status = "ok"
		slow = false
	} else if progress >= 3 {
		dbMs = 640
	}
	payMs := dbMs + 140
	checkMs := payMs + 40
	return []model.Trace{{
		ID:      "trace-142",
		TraceID: "8af2142c91",
		Clock:   "14:33",
		Spans: []model.Span{
			{ID: "span-1", Service: "checkout-api", Name: "POST /v1/checkout", DurationMs: checkMs, Status: status},
			{ID: "span-2", ParentID: "span-1", Service: "payment-api", Name: "POST /v1/charges", DurationMs: payMs, Status: status},
			{ID: "span-3", ParentID: "span-2", Service: "postgres", Name: "INSERT payment_intents", DurationMs: dbMs, Status: status, Slow: slow},
		},
	}}
}

func paymentEvents(progress int) []model.KubeEvent {
	events := []model.KubeEvent{
		{Clock: "14:31", Type: "Normal", Reason: "ScalingReplicaSet", Object: "deployment/payment-api", Message: "Scaled up replica set payment-api-7f9c8d to 8"},
		{Clock: "14:31", Type: "Normal", Reason: "DeploymentRollout", Object: "deployment/payment-api", Message: "Rolled out revision 48 (payment-api:v1.8.2)"},
		{Clock: "14:33", Type: "Warning", Reason: "Unhealthy", Object: "pod/payment-api-7f9c8d-xk2m", Message: "Readiness probe warning: request latency exceeded 1s"},
	}
	if progress >= 2 {
		events = append(events, model.KubeEvent{
			Clock: "14:40", Type: "Normal", Reason: "DeploymentRollback", Object: "deployment/payment-api",
			Message: "Rolled back to revision 47 (payment-api:v1.8.1)",
		})
	}
	return events
}

func (s *Store) namespaces(now time.Time) []model.Namespace {
	pay := paymentMetrics(s.completedSteps(now))
	payments := pay.status
	return []model.Namespace{
		{Name: "payments", Services: 1, Status: payments},
		{Name: "commerce", Services: 2, Status: model.StatusHealthy},
		{Name: "supply", Services: 1, Status: model.StatusHealthy},
		{Name: "platform", Services: 1, Status: model.StatusHealthy},
		{Name: "observability", Services: 0, Status: model.StatusHealthy},
	}
}

func nodes() []model.Node {
	return []model.Node{
		{Name: "node-a", Status: "Ready", Role: "worker", CPUPercent: 41, MemoryPercent: 58, Pods: 26, PodCapacity: 40, Zone: "lab-a", KubeletVersion: "v1.31.4"},
		{Name: "node-b", Status: "Ready", Role: "worker", CPUPercent: 36, MemoryPercent: 61, Pods: 22, PodCapacity: 40, Zone: "lab-a", KubeletVersion: "v1.31.4"},
		{Name: "node-c", Status: "Ready", Role: "worker", CPUPercent: 29, MemoryPercent: 49, Pods: 18, PodCapacity: 40, Zone: "lab-b", KubeletVersion: "v1.31.4"},
		{Name: "node-d", Status: "Ready", Role: "worker", CPUPercent: 33, MemoryPercent: 54, Pods: 21, PodCapacity: 40, Zone: "lab-b", KubeletVersion: "v1.31.4"},
	}
}

func controlPlane() []model.Component {
	return []model.Component{
		{Name: "kube-apiserver", Status: model.StatusHealthy},
		{Name: "etcd", Status: model.StatusHealthy},
		{Name: "kube-scheduler", Status: model.StatusHealthy},
		{Name: "kube-controller-manager", Status: model.StatusHealthy},
	}
}

func (s *Store) catalog(now time.Time) model.ExperimentCatalog {
	runs := make([]model.Experiment, 0, len(s.experiments))
	for _, run := range s.experiments {
		runs = append(runs, s.experimentView(run, now))
	}
	return model.ExperimentCatalog{
		Simulated: true,
		Notice:    "Experiments in this MVP are simulated. No pods are killed and no cluster credentials are used.",
		Scenarios: append([]model.Scenario(nil), scenarios...),
		Runs:      runs,
	}
}

func (s *Store) experimentView(run storedExperiment, now time.Time) model.Experiment {
	name := run.service
	for _, spec := range serviceSpecs {
		if spec.id == run.service {
			name = spec.name
		}
	}
	scenarioName := run.scenario
	if sc := scenarioByID(run.scenario); sc != nil {
		scenarioName = sc.Name
	}
	ends := run.started.Add(run.duration)
	status := "running"
	if !now.Before(ends) {
		status = "completed"
	}
	return model.Experiment{
		ID:           run.id,
		ServiceID:    run.service,
		ServiceName:  name,
		Scenario:     run.scenario,
		ScenarioName: scenarioName,
		DurationSec:  int(run.duration.Seconds()),
		Status:       status,
		Simulated:    true,
		StartedAt:    run.started,
		EndsAt:       ends,
		Note:         "Simulated experiment. Cluster state was not modified.",
	}
}

func scenarioByID(id string) *model.Scenario {
	for i := range scenarios {
		if scenarios[i].ID == id {
			return &scenarios[i]
		}
	}
	return nil
}

var scenarios = []model.Scenario{
	{ID: "kill-pod", Name: "Kill Pod", Description: "Terminate one ready pod and observe replacement.", Target: "pod"},
	{ID: "cpu-saturation", Name: "CPU Saturation", Description: "Pin a container near its CPU limit.", Target: "cpu"},
	{ID: "memory-pressure", Name: "Memory Pressure", Description: "Push a container toward its memory limit.", Target: "memory"},
	{ID: "network-latency", Name: "Network Latency", Description: "Add delay on the service's egress path.", Target: "network"},
	{ID: "dependency-failure", Name: "Dependency Failure", Description: "Fail calls to one downstream dependency.", Target: "dependency"},
	{ID: "database-connection-exhaustion", Name: "Database Connection Exhaustion", Description: "Hold database connections until the pool is saturated.", Target: "database"},
	{ID: "bad-deployment", Name: "Bad Deployment", Description: "Roll a known-bad version and watch the SLO.", Target: "deployment"},
}

func lerp(a, b, t float64) float64 {
	return a + (b-a)*t
}

func round(v float64, places int) float64 {
	p := math.Pow(10, float64(places))
	return math.Round(v*p) / p
}

func rank(status string) int {
	switch status {
	case model.StatusCritical:
		return 3
	case model.StatusDegraded:
		return 2
	default:
		return 1
	}
}
