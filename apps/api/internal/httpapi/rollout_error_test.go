package httpapi

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/opspilot/opspilot/apps/api/internal/executor"
	"github.com/opspilot/opspilot/apps/api/internal/repository/memory"
	"github.com/opspilot/opspilot/apps/api/internal/service"
)

func TestRolloutPolicyErrorIsBadRequest(t *testing.T) {
	epoch := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	store := memory.New(memory.Options{Epoch: epoch, Now: func() time.Time { return epoch }})
	svc := service.New(store, executor.Simulated{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	api := New(svc, slog.New(slog.NewTextHandler(io.Discard, nil)), "test")
	rec := httptest.NewRecorder()
	api.writeRolloutErr(rec, errors.New("rollout is not awaiting approval"))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
}
