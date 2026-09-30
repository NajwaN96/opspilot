package httpapi

import (
	"net/http"

	"github.com/opspilot/opspilot/apps/api/internal/buildmeta"
	"github.com/opspilot/opspilot/apps/api/internal/cloudstatus"
)

func (s *Server) cloudStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, cloudstatus.Current(r.Context()))
}

func (s *Server) release(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, buildmeta.Public("local-live"))
}
