package httpapi

import (
	"net/http"
	"regexp"
	"time"

	"github.com/opspilot/opspilot/apps/api/internal/detection"
	"github.com/opspilot/opspilot/apps/api/internal/telemetry"
)

var serviceIDPattern = regexp.MustCompile(`^k8s_[a-z0-9_-]+$`)

func (s *Server) telemetryStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.componentStatus(r))
}

func (s *Server) componentStatus(r *http.Request) map[string]string {
	if s.Sources == nil {
		return map[string]string{"prometheus": "unavailable", "opentelemetry": "unavailable", "traces": "unavailable"}
	}
	return s.Sources.ComponentStatus(r.Context())
}

func (s *Server) serviceTelemetry(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !serviceIDPattern.MatchString(id) {
		writeAPIError(w, http.StatusBadRequest, "invalid", "service id is not allowed")
		return
	}
	svc, err := s.svc.GetService(r.Context(), id)
	if err != nil {
		s.writeErr(w, err)
		return
	}
	body := telemetryBody{Source: "unavailable", Service: svc.Name, Namespace: svc.Namespace, Message: "Telemetry unavailable", DataSource: "none"}
	if svc.Source != "kubernetes" || svc.Namespace != "demo-shop" || s.Sources == nil {
		if svc.Source != "kubernetes" {
			body.Source = "simulated"
			body.Message = "This service uses the simulated incident plane."
		}
		writeJSON(w, http.StatusOK, body)
		return
	}
	snap, err := s.Sources.Prometheus.ServiceWindow(r.Context(), svc.Name, svc.Namespace, "1m")
	if err != nil || !snap.Available {
		writeJSON(w, http.StatusOK, body)
		return
	}
	body = telemetryBody{
		Source: "prometheus", Available: true, Service: svc.Name, Namespace: svc.Namespace,
		Updated: snap.Updated, RequestRate: snap.RequestRate, ErrorRate: snap.ErrorRate,
		Requests: snap.Requests, P50LatencyMs: snap.P50 * 1000, P95LatencyMs: snap.P95 * 1000,
		P99LatencyMs: snap.P99 * 1000, Availability: snap.Availability, DataSource: "Prometheus",
	}
	if svc.Name == detection.Service {
		wide, err := s.Sources.Prometheus.ServiceWindow(r.Context(), svc.Name, svc.Namespace, "15m")
		if err == nil && wide.Available {
			slo := detection.ComputeSLO(wide.Requests, wide.Errors)
			body.SLO = &sloBody{
				Target: slo.Target * 100, Window: "15m", Label: "Local observation window",
				Availability: slo.Availability, ErrorBudgetConsumed: slo.ErrorBudgetConsumed,
				BurnRate: slo.BurnRate, Sufficient: slo.Sufficient, Source: "Prometheus",
			}
		}
	}
	writeJSON(w, http.StatusOK, body)
}

func (s *Server) serviceTraces(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !serviceIDPattern.MatchString(id) {
		writeAPIError(w, http.StatusBadRequest, "invalid", "service id is not allowed")
		return
	}
	svc, err := s.svc.GetService(r.Context(), id)
	if err != nil {
		s.writeErr(w, err)
		return
	}
	if svc.Source != "kubernetes" || svc.Namespace != "demo-shop" || s.Sources == nil {
		writeJSON(w, http.StatusOK, telemetry.TraceList{Source: "unavailable", Message: "Telemetry unavailable", Traces: []telemetry.TraceSummary{}})
		return
	}
	list, err := s.Sources.Jaeger.Recent(r.Context(), svc.Name)
	if err != nil || list.Traces == nil {
		list.Traces = []telemetry.TraceSummary{}
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) traceDetail(w http.ResponseWriter, r *http.Request) {
	if s.Sources == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "unavailable", "Telemetry unavailable")
		return
	}
	item, err := s.Sources.Jaeger.Trace(r.Context(), r.PathValue("id"))
	if err != nil {
		writeAPIError(w, http.StatusNotFound, "not_found", "trace is unavailable")
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) stopExperiment(w http.ResponseWriter, r *http.Request) {
	item, err := s.svc.StopExperiment(r.Context(), r.PathValue("id"))
	if err != nil {
		s.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

type telemetryBody struct {
	Source       string    `json:"source"`
	Available    bool      `json:"available"`
	Message      string    `json:"message,omitempty"`
	Service      string    `json:"service"`
	Namespace    string    `json:"namespace"`
	Updated      time.Time `json:"updated,omitempty"`
	RequestRate  float64   `json:"requestRate"`
	ErrorRate    float64   `json:"errorRate"`
	Requests     float64   `json:"requests"`
	P50LatencyMs float64   `json:"p50LatencyMs"`
	P95LatencyMs float64   `json:"p95LatencyMs"`
	P99LatencyMs float64   `json:"p99LatencyMs"`
	Availability float64   `json:"availability"`
	DataSource   string    `json:"dataSource"`
	SLO          *sloBody  `json:"slo,omitempty"`
}

type sloBody struct {
	Target              float64 `json:"target"`
	Window              string  `json:"window"`
	Label               string  `json:"label"`
	Availability        float64 `json:"availability"`
	ErrorBudgetConsumed float64 `json:"errorBudgetConsumed"`
	BurnRate            float64 `json:"burnRate"`
	Sufficient          bool    `json:"sufficient"`
	Source              string  `json:"source"`
}
