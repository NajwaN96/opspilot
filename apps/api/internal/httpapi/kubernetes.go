package httpapi

import (
	"errors"
	"net/http"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/opspilot/opspilot/apps/api/internal/kubernetes"
)

func (s *Server) reader() kubernetes.Reader {
	if s.Kubernetes != nil {
		return s.Kubernetes
	}
	return kubernetes.Disconnected{Cluster: "opspilot-dev", Namespace: "demo-shop", Scope: []string{"demo-shop"}}
}

func (s *Server) kubernetesStatus(w http.ResponseWriter, r *http.Request) {
	item, err := s.reader().Status(r.Context())
	if err != nil {
		s.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) kubernetesNamespaces(w http.ResponseWriter, r *http.Request) {
	items, err := s.reader().Namespaces(r.Context())
	if err != nil {
		s.writeKubeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) kubernetesWorkloads(w http.ResponseWriter, r *http.Request) {
	items, err := s.reader().Workloads(r.Context())
	if err != nil {
		s.writeKubeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) kubernetesWorkload(w http.ResponseWriter, r *http.Request) {
	item, err := s.reader().Workload(r.Context(), r.PathValue("namespace"), r.PathValue("name"))
	if err != nil {
		s.writeKubeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) kubernetesPods(w http.ResponseWriter, r *http.Request) {
	items, err := s.reader().Pods(r.Context())
	if err != nil {
		s.writeKubeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) kubernetesEvents(w http.ResponseWriter, r *http.Request) {
	items, err := s.reader().Events(r.Context())
	if err != nil {
		s.writeKubeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) resetDemo(w http.ResponseWriter, r *http.Request) {
	if err := s.svc.ResetDemo(r.Context()); err != nil {
		s.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"status":   "reset",
		"incident": "INC-142",
		"scope":    "demo",
	})
}

func (s *Server) writeKubeErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, kubernetes.ErrDisconnected):
		writeAPIError(w, http.StatusServiceUnavailable, "disconnected", "Kubernetes cluster is disconnected")
	case apierrors.IsNotFound(err):
		writeAPIError(w, http.StatusNotFound, "not_found", "workload not found in the discovery scope")
	default:
		s.logger.Error("kubernetes read failed", "error", err)
		writeAPIError(w, http.StatusBadGateway, "kubernetes", "Kubernetes read failed")
	}
}
