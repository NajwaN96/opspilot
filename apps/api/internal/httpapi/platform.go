package httpapi

import (
	"net/http"
	"os"

	"github.com/opspilot/opspilot/apps/api/internal/platform"
)

func (s *Server) platformCatalog(w http.ResponseWriter, _ *http.Request) {
	catalog, _, err := s.platformData()
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "platform catalog unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"source":   "service-contract",
		"venue":    s.platformVenue(),
		"services": catalog.Services,
	})
}

func (s *Server) platformService(w http.ResponseWriter, r *http.Request) {
	catalog, _, err := s.platformData()
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "platform catalog unavailable"})
		return
	}
	service, ok := catalog.Get(r.PathValue("name"))
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "service is not in the catalog"})
		return
	}
	writeJSON(w, http.StatusOK, service)
}

func (s *Server) platformRunbooks(w http.ResponseWriter, _ *http.Request) {
	_, library, err := s.platformData()
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "runbook library unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, library)
}

func (s *Server) platformRunbook(w http.ResponseWriter, r *http.Request) {
	_, library, err := s.platformData()
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "runbook library unavailable"})
		return
	}
	book, ok := library.Get(r.PathValue("id"))
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "runbook not found"})
		return
	}
	writeJSON(w, http.StatusOK, book)
}

func (s *Server) platformScorecard(w http.ResponseWriter, _ *http.Request) {
	catalog, _, err := s.platformData()
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "platform catalog unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, platform.Scorecard(catalog))
}

func (s *Server) platformPreview(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"mode":       "preview",
		"writes":     false,
		"localWrite": s.platformVenue() == "local-live",
		"runtimes":   []string{"go", "node", "python"},
		"strategies": []string{"rolling", "canary"},
		"message":    "Preview does not create a repository or infrastructure. A local write stays under generated/services and refuses existing demo-shop services.",
	})
}

func (s *Server) platformGenerate(w http.ResponseWriter, r *http.Request) {
	if s.platformVenue() != "local-live" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "golden path writes are local only"})
		return
	}
	var req platform.GenerateRequest
	if err := decode(w, r, &req); err != nil {
		return
	}
	root, err := platform.FindRoot()
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "platform catalog unavailable"})
		return
	}
	catalog, _, err := platform.Load(root)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "platform catalog unavailable"})
		return
	}
	rel, err := platform.Materialize(root, req, catalog)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"path": rel, "mode": "local-preview"})
}

func (s *Server) platformData() (platform.Catalog, platform.Library, error) {
	root, err := platform.FindRoot()
	if err != nil {
		return platform.Catalog{}, platform.Library{}, err
	}
	return platform.Load(root)
}

func (s *Server) platformVenue() string {
	if os.Getenv("OPSPILOT_ENV") == "production" {
		return "aws-portfolio-demo"
	}
	return "local-live"
}
