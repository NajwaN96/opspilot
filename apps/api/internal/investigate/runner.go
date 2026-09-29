package investigate

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/opspilot/opspilot/apps/api/internal/detection"
	"github.com/opspilot/opspilot/apps/api/internal/model"
	"github.com/opspilot/opspilot/apps/api/internal/telemetry"
)

var durationPattern = regexp.MustCompile(`duration (\d+)ms`)

// Reader is the read-only evidence surface. It cannot mutate a cluster.
type Reader struct {
	Metrics interface {
		ServiceWindow(ctx context.Context, service, namespace, window string) (telemetry.Snapshot, error)
	}
	Traces interface {
		Recent(ctx context.Context, service string) (telemetry.TraceList, error)
		Trace(ctx context.Context, id string) (telemetry.TraceDetail, error)
	}
	Cluster interface {
		Workload(ctx context.Context, namespace, name string) (model.Workload, error)
		Pods(ctx context.Context) ([]model.Pod, error)
		Events(ctx context.Context) ([]model.ClusterEvent, error)
	}
}

type Record struct {
	ID              string
	IncidentID      string
	SnapshotID      string
	SnapshotVersion int
	Provider        string
	Model           string
	Status          string
	Output          Output
	Validation      Validation
	ErrorCategory   string
	Usage           Usage
	StartedAt       time.Time
	CompletedAt     *time.Time
	Real            bool
}

// Store persists snapshots and investigations. It is not a general SQL client.
type Store interface {
	OpenSubjects(ctx context.Context) ([]Subject, error)
	Subject(ctx context.Context, id string) (Subject, error)
	LatestSnapshot(ctx context.Context, incidentID string) (Snapshot, bool, error)
	SaveSnapshot(ctx context.Context, snapshot Snapshot) error
	LatestInvestigation(ctx context.Context, incidentID string) (Record, bool, error)
	BeginInvestigation(ctx context.Context, record Record) (bool, error)
	FinishInvestigation(ctx context.Context, record Record) error
	AppendIncidentEvent(ctx context.Context, incidentID, title, detail, kind string) error
	MergeTraces(ctx context.Context, incidentID string, traces []model.Trace) error
	SnapshotByID(ctx context.Context, id string) (Snapshot, bool, error)
}

type Runner struct {
	Provider     Provider
	Reader       Reader
	Store        Store
	Logger       *slog.Logger
	EnrichWindow time.Duration
	Now          func() time.Time
	mu           sync.Mutex
	running      map[string]bool
	statusValue  string
	statusAt     time.Time
}

func (r *Runner) now() time.Time {
	if r.Now != nil {
		return r.Now()
	}
	return time.Now().UTC()
}

func (r *Runner) window() time.Duration {
	if r.EnrichWindow > 0 {
		return r.EnrichWindow
	}
	return 45 * time.Second
}

// Tick enriches open incidents and starts at most one investigation per snapshot.
func (r *Runner) Tick(ctx context.Context) error {
	if r.Store == nil || r.Provider == nil {
		return nil
	}
	subjects, err := r.Store.OpenSubjects(ctx)
	if err != nil {
		return err
	}
	for _, subject := range subjects {
		if err := r.Consider(ctx, subject.ID, false); err != nil && r.Logger != nil {
			r.Logger.Error("investigation", "incident", subject.ID, "error", err.Error())
		}
	}
	return nil
}

// Consider builds or enriches evidence, then investigates once.
func (r *Runner) Consider(ctx context.Context, incidentID string, manual bool) error {
	if r.Store == nil || r.Provider == nil {
		return errors.New("ai investigation unavailable")
	}
	r.mu.Lock()
	if r.running == nil {
		r.running = map[string]bool{}
	}
	if r.running[incidentID] {
		r.mu.Unlock()
		return nil
	}
	r.running[incidentID] = true
	r.mu.Unlock()
	defer func() {
		r.mu.Lock()
		delete(r.running, incidentID)
		r.mu.Unlock()
	}()

	subject, err := r.Store.Subject(ctx, incidentID)
	if err != nil {
		return err
	}
	snapshot, found, err := r.Store.LatestSnapshot(ctx, incidentID)
	if err != nil {
		return err
	}
	if !found {
		snapshot = r.collect(ctx, subject, nil)
		snapshot.ID = newID("ev-")
		snapshot.Version = 1
		if err := r.Store.SaveSnapshot(ctx, snapshot); err != nil {
			return err
		}
		_ = r.Store.AppendIncidentEvent(ctx, incidentID, "Evidence snapshot created", fmt.Sprintf("version %d", snapshot.Version), "evidence")
		_ = r.persistTraces(ctx, incidentID, snapshot)
	}
	enriched := r.enrich(ctx, subject, snapshot)
	if enriched.TraceCount > snapshot.TraceCount {
		enriched.ID = newID("ev-")
		enriched.Version = snapshot.Version + 1
		if err := r.Store.SaveSnapshot(ctx, enriched); err != nil {
			return err
		}
		_ = r.Store.AppendIncidentEvent(ctx, incidentID, "Trace evidence enriched", fmt.Sprintf("version %d traces %d", enriched.Version, enriched.TraceCount), "evidence")
		_ = r.persistTraces(ctx, incidentID, enriched)
		snapshot = enriched
	}
	if !manual && snapshot.TraceCount == 0 && r.now().Sub(subject.Started) < r.window() {
		return nil
	}
	return r.investigate(ctx, subject, snapshot, manual)
}

func (r *Runner) investigate(ctx context.Context, subject Subject, snapshot Snapshot, manual bool) error {
	existing, found, err := r.Store.LatestInvestigation(ctx, subject.ID)
	if err != nil {
		return err
	}
	if found && existing.SnapshotVersion == snapshot.Version {
		switch existing.Status {
		case StatusQueued, StatusCollecting, StatusInvestigating, StatusValidating, StatusCompleted:
			return nil
		case StatusFailed, StatusInvalid:
			if !manual {
				return nil
			}
		}
	}
	if !r.Provider.Real() && r.Provider.Name() == "disabled" {
		return r.fail(ctx, subject, snapshot, "disabled", "ai investigation unavailable")
	}
	record := Record{
		ID: newID("ai-"), IncidentID: subject.ID, SnapshotID: snapshot.ID, SnapshotVersion: snapshot.Version,
		Provider: r.Provider.Name(), Model: r.Provider.Model(), Status: StatusInvestigating,
		StartedAt: r.now(), Real: false,
	}
	started, err := r.Store.BeginInvestigation(ctx, record)
	if err != nil || !started {
		return err
	}
	_ = r.Store.AppendIncidentEvent(ctx, subject.ID, "AI investigation started", r.Provider.Name()+" "+r.Provider.Model(), "investigation")
	output, usage, err := r.Provider.Investigate(ctx, snapshot)
	record.Usage = usage
	if err != nil {
		return r.failRecord(ctx, record, category(err))
	}
	raw, err := json.Marshal(output)
	if err != nil {
		return r.fail(ctx, subject, snapshot, "invalid", "invalid output")
	}
	checked, validation := Validate(snapshot, raw)
	record.Output = checked
	record.Validation = validation
	now := r.now()
	record.CompletedAt = &now
	record.Real = false
	if validation.Accepted {
		record.Status = StatusCompleted
		record.Real = r.Provider.Real()
	} else {
		record.Status = StatusInvalid
	}
	if err := r.Store.FinishInvestigation(ctx, record); err != nil {
		return err
	}
	title := "AI investigation completed"
	if !validation.Accepted {
		title = "AI investigation failed"
	}
	_ = r.Store.AppendIncidentEvent(ctx, subject.ID, title, record.Status, "investigation")
	if validation.Accepted {
		_ = r.Store.AppendIncidentEvent(ctx, subject.ID, "AI investigation validated", "citations match the evidence snapshot", "investigation")
		_ = r.Store.AppendIncidentEvent(ctx, subject.ID, "AI recommendation generated", checked.RecommendedAction.ActionType+" is not executable", "investigation")
	}
	return nil
}

func (r *Runner) failRecord(ctx context.Context, record Record, categoryName string) error {
	now := r.now()
	record.Status = StatusFailed
	record.Real = false
	record.ErrorCategory = categoryName
	record.CompletedAt = &now
	if err := r.Store.FinishInvestigation(ctx, record); err != nil {
		return err
	}
	_ = r.Store.AppendIncidentEvent(ctx, record.IncidentID, "AI investigation failed", categoryName, "investigation")
	if r.Logger != nil {
		r.Logger.Error("ai investigation failed", "incident", record.IncidentID, "category", categoryName)
	}
	return fmt.Errorf("%s", categoryName)
}

func (r *Runner) fail(ctx context.Context, subject Subject, snapshot Snapshot, categoryName, detail string) error {
	now := r.now()
	record := Record{
		ID: newID("ai-"), IncidentID: subject.ID, SnapshotID: snapshot.ID, SnapshotVersion: snapshot.Version,
		Provider: r.Provider.Name(), Model: r.Provider.Model(), Status: StatusFailed,
		ErrorCategory: categoryName, StartedAt: now, CompletedAt: &now, Real: false,
	}
	if existing, found, _ := r.Store.LatestInvestigation(ctx, subject.ID); found && existing.SnapshotVersion == snapshot.Version && existing.Status == StatusInvestigating {
		record.ID = existing.ID
		_ = r.Store.FinishInvestigation(ctx, record)
	} else {
		started, err := r.Store.BeginInvestigation(ctx, record)
		if err != nil {
			return err
		}
		if started {
			record.Status = StatusFailed
			_ = r.Store.FinishInvestigation(ctx, record)
		}
	}
	_ = r.Store.AppendIncidentEvent(ctx, subject.ID, "AI investigation failed", categoryName, "investigation")
	if r.Logger != nil {
		r.Logger.Error("ai investigation failed", "incident", subject.ID, "category", categoryName)
	}
	_ = detail
	return fmt.Errorf("%s", categoryName)
}

func (r *Runner) collect(ctx context.Context, subject Subject, prior *Snapshot) Snapshot {
	obs := Observations{Now: r.now()}
	if r.Reader.Metrics != nil {
		if snap, err := r.Reader.Metrics.ServiceWindow(ctx, detection.Service, detection.Namespace, "1m"); err == nil {
			obs.Metrics = snap
		}
		if snap, err := r.Reader.Metrics.ServiceWindow(ctx, detection.Service, detection.Namespace, "15m"); err == nil {
			obs.SLO = snap
		}
	}
	if r.Reader.Cluster != nil {
		if workload, err := r.Reader.Cluster.Workload(ctx, detection.Namespace, detection.Service); err == nil {
			obs.Workload = workload
			if subject.Analysis.LikelyCause == "" {
				subject.Facts.Version = workload.Version
				subject.Facts.Ready = workload.Ready
				subject.Facts.Desired = workload.Desired
				subject.Facts.Restarts = workload.Restarts
				finding := detection.Diagnose(subject.Facts)
				subject.Analysis.LikelyCause = finding.LikelyCause
				subject.Analysis.Supporting = finding.Supporting
				subject.Analysis.Contradicting = finding.Contradicting
			}
		}
		if pods, err := r.Reader.Cluster.Pods(ctx); err == nil {
			for _, pod := range pods {
				if pod.Namespace == detection.Namespace && pod.Service == detection.Service {
					obs.Pods = append(obs.Pods, pod)
				}
			}
		}
		if events, err := r.Reader.Cluster.Events(ctx); err == nil {
			for _, event := range events {
				if event.Namespace == detection.Namespace {
					obs.Events = append(obs.Events, event)
				}
			}
		}
	}
	obs.Traces = r.traces(ctx)
	if prior != nil && len(obs.Traces) == 0 {
		return *prior
	}
	snapshot := Build(subject, obs)
	snapshot.IncidentID = subject.ID
	return snapshot
}

func (r *Runner) enrich(ctx context.Context, subject Subject, current Snapshot) Snapshot {
	if current.TraceCount > 0 {
		return current
	}
	next := r.collect(ctx, subject, &current)
	if next.TraceCount == 0 {
		return current
	}
	return next
}

func (r *Runner) traces(ctx context.Context) []telemetry.TraceDetail {
	if r.Reader.Traces == nil {
		return nil
	}
	list, err := r.Reader.Traces.Recent(ctx, detection.Service)
	if err != nil || list.Source != "opentelemetry" {
		return nil
	}
	var out []telemetry.TraceDetail
	for _, id := range SelectTraces(list) {
		detail, err := r.Reader.Traces.Trace(ctx, id)
		if err != nil || detail.ID == "" {
			continue
		}
		out = append(out, detail)
	}
	return out
}

func (r *Runner) persistTraces(ctx context.Context, incidentID string, snapshot Snapshot) error {
	var traces []model.Trace
	for _, item := range snapshot.Items {
		if item.Kind != "trace" {
			continue
		}
		status := "ok"
		if strings.Contains(item.Body, "status error") {
			status = "error"
		}
		duration := 0
		if match := durationPattern.FindStringSubmatch(item.Body); len(match) == 2 {
			fmt.Sscan(match[1], &duration)
		}
		traces = append(traces, model.Trace{
			ID: item.Title, TraceID: item.Title, Clock: item.At.UTC().Format(time.RFC3339),
			Spans: []model.Span{{ID: item.ID, Name: "payment path", Service: "payment-api", Status: status, DurationMs: duration, Slow: duration >= 300}},
		})
	}
	if len(traces) == 0 {
		return nil
	}
	return r.Store.MergeTraces(ctx, incidentID, traces)
}

// View returns the latest investigation and the snapshot it used.
func (r *Runner) View(ctx context.Context, incidentID string) (View, error) {
	view := View{Status: StatusNotRequested, Provider: r.Provider.Name(), Model: r.Provider.Model()}
	if r.Provider == nil || r.Provider.Name() == "disabled" {
		view.Status = "unavailable"
		view.Error = "ai investigation unavailable"
		view.Provider = "disabled"
	}
	if r.Store == nil {
		return view, nil
	}
	snapshot, found, err := r.Store.LatestSnapshot(ctx, incidentID)
	if err != nil {
		return view, err
	}
	if found {
		view.SnapshotID = snapshot.ID
		view.SnapshotVersion = snapshot.Version
		view.Evidence = snapshot.Items
		view.Omitted = snapshot.Omitted
	}
	record, found, err := r.Store.LatestInvestigation(ctx, incidentID)
	if err != nil || !found {
		return view, err
	}
	view.Status = record.Status
	view.Provider = record.Provider
	view.Model = record.Model
	view.Real = record.Real && record.Provider == "openai" && record.Status == StatusCompleted && record.Validation.Accepted
	view.SnapshotID = record.SnapshotID
	view.SnapshotVersion = record.SnapshotVersion
	view.Summary = record.Output.Summary
	view.LikelyCauses = record.Output.LikelyCauses
	view.Observations = record.Output.Observations
	view.RecommendedAction = record.Output.RecommendedAction
	view.MissingInformation = record.Output.MissingInformation
	view.RiskNotes = record.Output.RiskNotes
	view.Validation = record.Validation
	view.Error = record.ErrorCategory
	if !record.StartedAt.IsZero() {
		started := record.StartedAt
		view.StartedAt = &started
	}
	view.CompletedAt = record.CompletedAt
	if snap, ok, err := r.snapshotByID(ctx, incidentID, record.SnapshotID); err == nil && ok {
		view.Evidence = snap.Items
		view.Omitted = snap.Omitted
	}
	return view, nil
}

func (r *Runner) snapshotByID(ctx context.Context, _, id string) (Snapshot, bool, error) {
	if id == "" {
		return Snapshot{}, false, nil
	}
	return r.Store.SnapshotByID(ctx, id)
}

func (r *Runner) Status(ctx context.Context) string {
	r.mu.Lock()
	if r.statusValue != "" && time.Since(r.statusAt) < time.Minute {
		value := r.statusValue
		r.mu.Unlock()
		return value
	}
	r.mu.Unlock()
	value := "disabled"
	if r.Provider != nil {
		value = r.Provider.Status(ctx)
	}
	r.mu.Lock()
	r.statusValue = value
	r.statusAt = time.Now()
	r.mu.Unlock()
	return value
}

func category(err error) string {
	if err == nil {
		return ""
	}
	text := err.Error()
	switch {
	case text == "unauthorized":
		return "unauthorized"
	case strings.HasPrefix(text, "quota"):
		return text
	case text == "unconfigured" || text == "ai investigation unavailable":
		return "unconfigured"
	case text == "invalid output":
		return "invalid"
	default:
		return "unavailable"
	}
}

func newID(prefix string) string {
	var buf [4]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return prefix + "00000000"
	}
	return prefix + hex.EncodeToString(buf[:])
}
