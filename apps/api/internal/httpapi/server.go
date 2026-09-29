package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/opspilot/opspilot/apps/api/internal/investigate"
	"github.com/opspilot/opspilot/apps/api/internal/kubernetes"
	"github.com/opspilot/opspilot/apps/api/internal/release"
	"github.com/opspilot/opspilot/apps/api/internal/repository"
	"github.com/opspilot/opspilot/apps/api/internal/rollout"
	"github.com/opspilot/opspilot/apps/api/internal/service"
	"github.com/opspilot/opspilot/apps/api/internal/telemetry"
)

type Server struct {
	svc     *service.Service
	logger  *slog.Logger
	version string
	// Kubernetes is read-only. A nil reader is reported as disconnected.
	Kubernetes kubernetes.Reader
	// Ready reports dependencies. Nil means the process has no external dependencies.
	Ready func(r *http.Request) map[string]any
	// Sources reads local Prometheus and Jaeger. It does not accept browser PromQL.
	Sources *telemetry.Sources
	// Investigator reads prepared evidence. It cannot mutate Kubernetes.
	Investigator *investigate.Runner
	// Rollouts is the canary engine. It is not a generic Kubernetes client.
	Rollouts *rollout.Engine
	AllowLab bool
}

func New(svc *service.Service, logger *slog.Logger, version string) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	return &Server{svc: svc, logger: logger, version: version}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("GET /ready", s.ready)
	mux.HandleFunc("GET /api/v1/clusters", s.listClusters)
	mux.HandleFunc("GET /api/v1/services", s.listServices)
	mux.HandleFunc("GET /api/v1/services/{id}", s.getService)
	mux.HandleFunc("GET /api/v1/services/{id}/telemetry", s.serviceTelemetry)
	mux.HandleFunc("GET /api/v1/services/{id}/traces", s.serviceTraces)
	mux.HandleFunc("GET /api/v1/traces/{id}", s.traceDetail)
	mux.HandleFunc("GET /api/v1/telemetry/status", s.telemetryStatus)
	mux.HandleFunc("GET /api/v1/incidents", s.listIncidents)
	mux.HandleFunc("GET /api/v1/incidents/{id}", s.getIncident)
	mux.HandleFunc("POST /api/v1/incidents/{id}/remediations", s.startRemediation)
	mux.HandleFunc("GET /api/v1/incidents/{id}/investigation", s.getInvestigation)
	mux.HandleFunc("POST /api/v1/incidents/{id}/investigation", s.runInvestigation)
	mux.HandleFunc("GET /api/v1/ai/status", s.aiStatus)
	mux.HandleFunc("GET /api/v1/rollouts", s.listRollouts)
	mux.HandleFunc("GET /api/v1/rollouts/{id}", s.getRollout)
	mux.HandleFunc("POST /api/v1/rollouts/{id}/approve", s.approveRollout)
	mux.HandleFunc("POST /api/v1/lab/payment-api/canary/good", s.startGoodCanary)
	mux.HandleFunc("POST /api/v1/lab/payment-api/canary/bad", s.startBadCanary)
	mux.HandleFunc("GET /metrics", s.rolloutMetrics)
	mux.HandleFunc("GET /api/v1/experiments", s.listExperiments)
	mux.HandleFunc("POST /api/v1/experiments", s.startExperiment)
	mux.HandleFunc("POST /api/v1/experiments/{id}/stop", s.stopExperiment)
	mux.HandleFunc("POST /api/v1/rollouts/payment-api/bad", s.deployBadPayment)
	mux.HandleFunc("GET /api/v1/kubernetes/status", s.kubernetesStatus)
	mux.HandleFunc("GET /api/v1/kubernetes/namespaces", s.kubernetesNamespaces)
	mux.HandleFunc("GET /api/v1/kubernetes/workloads", s.kubernetesWorkloads)
	mux.HandleFunc("GET /api/v1/kubernetes/workloads/{namespace}/{name}", s.kubernetesWorkload)
	mux.HandleFunc("GET /api/v1/kubernetes/pods", s.kubernetesPods)
	mux.HandleFunc("GET /api/v1/kubernetes/events", s.kubernetesEvents)
	mux.HandleFunc("POST /api/v1/demo/reset", s.resetDemo)
	mux.HandleFunc("GET /{$}", s.root)
	return s.logged(mux)
}

func (s *Server) root(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"service": "opspilot-api",
		"health":  "/health",
		"api":     "/api/v1",
	})
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"service": "opspilot-api",
		"version": s.version,
	})
}

func (s *Server) ready(w http.ResponseWriter, r *http.Request) {
	body := map[string]any{
		"status":           "ok",
		"database":         "memory",
		"kubernetes":       "disconnected",
		"demoResetEnabled": false,
	}
	if s.Ready != nil {
		for key, value := range s.Ready(r) {
			body[key] = value
		}
	}
	status := http.StatusOK
	if body["database"] == "down" {
		body["status"] = "unavailable"
		status = http.StatusServiceUnavailable
	}
	writeJSON(w, status, body)
}

func (s *Server) listClusters(w http.ResponseWriter, r *http.Request) {
	items, err := s.svc.ListClusters(r.Context())
	if err != nil {
		s.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) listServices(w http.ResponseWriter, r *http.Request) {
	items, err := s.svc.ListServices(r.Context())
	if err != nil {
		s.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) getService(w http.ResponseWriter, r *http.Request) {
	item, err := s.svc.GetService(r.Context(), r.PathValue("id"))
	if err != nil {
		s.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) listIncidents(w http.ResponseWriter, r *http.Request) {
	items, err := s.svc.ListIncidents(r.Context())
	if err != nil {
		s.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) getIncident(w http.ResponseWriter, r *http.Request) {
	item, err := s.svc.GetIncident(r.Context(), r.PathValue("id"))
	if err != nil {
		s.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) startRemediation(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Action string `json:"action"`
	}
	if err := decode(w, r, &body); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid", "request body must be JSON")
		return
	}
	item, err := s.svc.StartRemediation(r.Context(), r.PathValue("id"), body.Action)
	if err != nil {
		s.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, item)
}

func (s *Server) listExperiments(w http.ResponseWriter, r *http.Request) {
	item, err := s.svc.ListExperiments(r.Context())
	if err != nil {
		s.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) startExperiment(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ServiceID   string `json:"serviceId"`
		Scenario    string `json:"scenario"`
		DurationSec int    `json:"durationSec"`
	}
	if err := decode(w, r, &body); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid", "request body must be JSON")
		return
	}
	item, err := s.svc.StartExperiment(r.Context(), body.ServiceID, body.Scenario, body.DurationSec)
	if err != nil {
		s.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, item)
}

func (s *Server) getInvestigation(w http.ResponseWriter, r *http.Request) {
	if s.Investigator == nil {
		writeJSON(w, http.StatusOK, investigate.View{Status: "unavailable", Provider: "disabled", Error: "ai investigation unavailable"})
		return
	}
	view, err := s.Investigator.View(r.Context(), r.PathValue("id"))
	if err != nil {
		s.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (s *Server) runInvestigation(w http.ResponseWriter, r *http.Request) {
	if s.Investigator == nil {
		writeJSON(w, http.StatusOK, investigate.View{Status: "unavailable", Provider: "disabled", Error: "ai investigation unavailable"})
		return
	}
	var body struct{}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil && !errors.Is(err, io.EOF) {
		writeAPIError(w, http.StatusBadRequest, "invalid", "this action does not accept parameters")
		return
	}
	cerr := s.Investigator.Consider(r.Context(), r.PathValue("id"), true)
	view, err := s.Investigator.View(r.Context(), r.PathValue("id"))
	if err != nil {
		s.writeErr(w, err)
		return
	}
	if view.Status == investigate.StatusNotRequested && cerr != nil {
		s.writeErr(w, cerr)
		return
	}
	writeJSON(w, http.StatusAccepted, view)
}

func (s *Server) aiStatus(w http.ResponseWriter, r *http.Request) {
	body := map[string]string{"provider": "disabled", "model": "", "status": "disabled"}
	if s.Investigator != nil && s.Investigator.Provider != nil {
		body["provider"] = s.Investigator.Provider.Name()
		body["model"] = s.Investigator.Provider.Model()
		body["status"] = s.Investigator.Status(r.Context())
	}
	writeJSON(w, http.StatusOK, body)
}

func (s *Server) listRollouts(w http.ResponseWriter, r *http.Request) {
	if s.Rollouts == nil {
		writeJSON(w, http.StatusOK, []rollout.Rollout{})
		return
	}
	items, err := s.Rollouts.List(r.Context())
	if err != nil {
		s.writeRolloutErr(w, err)
		return
	}
	if items == nil {
		items = []rollout.Rollout{}
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) getRollout(w http.ResponseWriter, r *http.Request) {
	if s.Rollouts == nil {
		writeAPIError(w, http.StatusNotFound, "not_found", "rollout not found")
		return
	}
	view, err := s.Rollouts.View(r.Context(), r.PathValue("id"))
	if err != nil {
		s.writeRolloutErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (s *Server) approveRollout(w http.ResponseWriter, r *http.Request) {
	if s.Rollouts == nil {
		writeAPIError(w, http.StatusForbidden, "forbidden", "rollouts are not configured")
		return
	}
	var body struct {
		Action string `json:"action"`
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid", "this action does not accept parameters")
		return
	}
	view, err := s.Rollouts.Approve(r.Context(), r.PathValue("id"), body.Action)
	if err != nil {
		s.writeRolloutErr(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, view)
}

func (s *Server) startGoodCanary(w http.ResponseWriter, r *http.Request) {
	s.startCanary(w, r, "good")
}

func (s *Server) startBadCanary(w http.ResponseWriter, r *http.Request) {
	s.startCanary(w, r, "bad")
}

func (s *Server) startCanary(w http.ResponseWriter, r *http.Request, kind string) {
	if !s.AllowLab || s.Rollouts == nil {
		writeAPIError(w, http.StatusForbidden, "forbidden", "canary experiments are disabled")
		return
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	var body struct{}
	if err := dec.Decode(&body); err != nil && !errors.Is(err, io.EOF) {
		writeAPIError(w, http.StatusBadRequest, "invalid", "this action does not accept parameters")
		return
	}
	view, err := s.Rollouts.Start(r.Context(), kind)
	if err != nil {
		s.writeRolloutErr(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, view)
}

func (s *Server) rolloutMetrics(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	if s.Rollouts == nil || s.Rollouts.Counters == nil {
		_, _ = w.Write([]byte(""))
		return
	}
	_, _ = w.Write([]byte(s.Rollouts.Counters.Render()))
}

func (s *Server) writeRolloutErr(w http.ResponseWriter, err error) {
	text := err.Error()
	switch {
	case strings.Contains(text, "no rows") || strings.Contains(text, "missing"):
		writeAPIError(w, http.StatusNotFound, "not_found", "rollout not found")
	case strings.Contains(text, "already active"):
		writeAPIError(w, http.StatusConflict, "conflict", text)
	case strings.Contains(text, "must") || strings.Contains(text, "requires") || strings.Contains(text, "not ") || strings.Contains(text, "unknown") || strings.Contains(text, "does not"):
		writeAPIError(w, http.StatusBadRequest, "invalid", text)
	default:
		s.logger.Error("rollout", "error", err)
		writeAPIError(w, http.StatusInternalServerError, "internal", "internal error")
	}
}

func (s *Server) deployBadPayment(w http.ResponseWriter, r *http.Request) {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	var body struct{}
	if err := dec.Decode(&body); err != nil && !errors.Is(err, io.EOF) {
		writeAPIError(w, http.StatusBadRequest, "invalid", "this action does not accept parameters")
		return
	}
	if err := s.svc.DeployBadPayment(r.Context()); err != nil {
		s.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{
		"action":     "deploy-bad-payment",
		"cluster":    release.Cluster,
		"namespace":  release.Namespace,
		"deployment": release.Deployment,
		"version":    release.BadVersion,
	})
}

func (s *Server) writeErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		writeAPIError(w, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, repository.ErrConflict):
		writeAPIError(w, http.StatusConflict, "conflict", err.Error())
	case errors.Is(err, repository.ErrInvalid):
		writeAPIError(w, http.StatusBadRequest, "invalid", err.Error())
	case errors.Is(err, repository.ErrForbidden):
		writeAPIError(w, http.StatusForbidden, "forbidden", err.Error())
	default:
		s.logger.Error("request failed", "error", err)
		writeAPIError(w, http.StatusInternalServerError, "internal", "internal error")
	}
}

func decode(w http.ResponseWriter, r *http.Request, dest any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dest)
}

type apiError struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func writeAPIError(w http.ResponseWriter, status int, code, message string) {
	var body apiError
	body.Error.Code = code
	body.Error.Message = message
	writeJSON(w, status, body)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		slog.Error("encode response", "error", err)
	}
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(b)
}

func (s *Server) logged(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = newRequestID()
		}
		w.Header().Set("X-Request-ID", requestID)
		sw := &statusWriter{ResponseWriter: w}
		next.ServeHTTP(sw, r.WithContext(r.Context()))
		s.logger.Info("request",
			"request_id", requestID,
			"method", r.Method,
			"path", r.URL.Path,
			"status", sw.status,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}

func newRequestID() string {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(buf[:])
}
