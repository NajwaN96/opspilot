package httpapi_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/opspilot/opspilot/apps/api/internal/executor"
	"github.com/opspilot/opspilot/apps/api/internal/httpapi"
	"github.com/opspilot/opspilot/apps/api/internal/repository/memory"
	"github.com/opspilot/opspilot/apps/api/internal/service"
)

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	epoch := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	store := memory.New(memory.Options{
		Epoch: epoch,
		Now:   func() time.Time { return epoch },
		Step:  0,
	})
	svc := service.New(store, executor.Simulated{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	api := httpapi.New(svc, slog.New(slog.NewTextHandler(io.Discard, nil)), "test")
	return httptest.NewServer(api.Handler())
}

func TestHealth(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	res, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d", res.StatusCode)
	}
	var body map[string]string
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "ok" || body["service"] != "opspilot-api" {
		t.Fatalf("body %#v", body)
	}
}

func TestPaymentServiceIsCritical(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	res, err := http.Get(srv.URL + "/api/v1/services")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var services []map[string]any
	if err := json.NewDecoder(res.Body).Decode(&services); err != nil {
		t.Fatal(err)
	}
	var payment map[string]any
	for _, svc := range services {
		if svc["id"] == "payment-api" {
			payment = svc
		}
	}
	if payment == nil {
		t.Fatal("payment-api missing")
	}
	if payment["status"] != "critical" {
		t.Fatalf("status %v", payment["status"])
	}
	if payment["errorRate"] != 12.4 {
		t.Fatalf("errorRate %v", payment["errorRate"])
	}
	if payment["p95LatencyMs"] != float64(1800) {
		t.Fatalf("p95 %v", payment["p95LatencyMs"])
	}
	if payment["availability"] != 98.71 {
		t.Fatalf("availability %v", payment["availability"])
	}
	if payment["version"] != "v1.8.2" {
		t.Fatalf("version %v", payment["version"])
	}
}

func TestIncidentInvestigation(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	res, err := http.Get(srv.URL + "/api/v1/incidents/INC-142")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d", res.StatusCode)
	}
	var incident map[string]any
	if err := json.NewDecoder(res.Body).Decode(&incident); err != nil {
		t.Fatal(err)
	}
	if incident["title"] != "Payment API latency and elevated error rate" {
		t.Fatalf("title %v", incident["title"])
	}
	if incident["severity"] != "SEV-2" || incident["status"] != "investigating" {
		t.Fatalf("severity/status %v %v", incident["severity"], incident["status"])
	}
	analysis := incident["analysis"].(map[string]any)
	if analysis["simulated"] != true {
		t.Fatal("analysis should be marked simulated")
	}
	if analysis["confidence"] != 0.91 {
		t.Fatalf("confidence %v", analysis["confidence"])
	}
	if analysis["cause"] != "Database connection leak introduced by payment-api:v1.8.2" {
		t.Fatalf("cause %v", analysis["cause"])
	}
	timeline := incident["timeline"].([]any)
	first := timeline[0].(map[string]any)
	if first["clock"] != "14:31" || first["title"] != "Deployment payment-api:v1.8.2" {
		t.Fatalf("first event %#v", first)
	}
	evidence := incident["evidence"].(map[string]any)
	logs := evidence["logs"].([]any)
	foundPool := false
	for _, raw := range logs {
		line := raw.(map[string]any)
		if line["message"] == "database connection pool exhausted" {
			foundPool = true
		}
	}
	if !foundPool {
		t.Fatal("expected pool exhaustion log")
	}
	traces := evidence["traces"].([]any)
	spans := traces[0].(map[string]any)["spans"].([]any)
	var slow bool
	for _, raw := range spans {
		span := raw.(map[string]any)
		if span["service"] == "postgres" && span["slow"] == true {
			slow = true
		}
	}
	if !slow {
		t.Fatal("expected slow postgres span")
	}
}

func TestUnknownIncident(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()
	res, err := http.Get(srv.URL + "/api/v1/incidents/INC-404")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("status %d", res.StatusCode)
	}
}

func TestSimulatedRollbackResolvesIncident(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	body := bytes.NewBufferString(`{"action":"rollback"}`)
	res, err := http.Post(srv.URL+"/api/v1/incidents/INC-142/remediations", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusAccepted {
		t.Fatalf("status %d", res.StatusCode)
	}
	var remediation map[string]any
	if err := json.NewDecoder(res.Body).Decode(&remediation); err != nil {
		t.Fatal(err)
	}
	if remediation["simulated"] != true || remediation["status"] != "succeeded" {
		t.Fatalf("remediation %#v", remediation)
	}
	steps := remediation["steps"].([]any)
	if len(steps) != 5 {
		t.Fatalf("steps %d", len(steps))
	}
	last := steps[4].(map[string]any)
	if last["name"] != "Incident resolved" || last["status"] != "complete" {
		t.Fatalf("last step %#v", last)
	}

	incRes, err := http.Get(srv.URL + "/api/v1/incidents/INC-142")
	if err != nil {
		t.Fatal(err)
	}
	defer incRes.Body.Close()
	var incident map[string]any
	if err := json.NewDecoder(incRes.Body).Decode(&incident); err != nil {
		t.Fatal(err)
	}
	if incident["status"] != "resolved" {
		t.Fatalf("status %v", incident["status"])
	}
	snapshot := incident["snapshot"].(map[string]any)
	if snapshot["errorRate"] != 0.3 {
		t.Fatalf("errorRate %v", snapshot["errorRate"])
	}
	if snapshot["p95LatencyMs"] != float64(240) {
		t.Fatalf("p95 %v", snapshot["p95LatencyMs"])
	}
	if snapshot["dbConnections"] != float64(37) {
		t.Fatalf("db %v", snapshot["dbConnections"])
	}

	svcRes, err := http.Get(srv.URL + "/api/v1/services/payment-api")
	if err != nil {
		t.Fatal(err)
	}
	defer svcRes.Body.Close()
	var svc map[string]any
	if err := json.NewDecoder(svcRes.Body).Decode(&svc); err != nil {
		t.Fatal(err)
	}
	if svc["status"] != "healthy" || svc["version"] != "v1.8.1" {
		t.Fatalf("service status/version %v %v", svc["status"], svc["version"])
	}
}

func TestPolicyRejectsUnexpectedAction(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()
	body := bytes.NewBufferString(`{"action":"delete-namespace"}`)
	res, err := http.Post(srv.URL+"/api/v1/incidents/INC-142/remediations", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("status %d", res.StatusCode)
	}
}

func TestClusterAndExperiment(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	res, err := http.Get(srv.URL + "/api/v1/clusters")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var clusters []map[string]any
	if err := json.NewDecoder(res.Body).Decode(&clusters); err != nil {
		t.Fatal(err)
	}
	if len(clusters) != 1 || clusters[0]["name"] != "production-01" || clusters[0]["simulated"] != true {
		t.Fatalf("clusters %#v", clusters)
	}

	body := bytes.NewBufferString(`{"serviceId":"checkout-api","scenario":"kill-pod","durationSec":30}`)
	expRes, err := http.Post(srv.URL+"/api/v1/experiments", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer expRes.Body.Close()
	if expRes.StatusCode != http.StatusAccepted {
		t.Fatalf("status %d", expRes.StatusCode)
	}
	var experiment map[string]any
	if err := json.NewDecoder(expRes.Body).Decode(&experiment); err != nil {
		t.Fatal(err)
	}
	if experiment["simulated"] != true || experiment["scenario"] != "kill-pod" {
		t.Fatalf("experiment %#v", experiment)
	}
}

func TestReadyAndDemoResetGate(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	res, err := http.Get(srv.URL + "/ready")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.Header.Get("X-Request-ID") == "" {
		t.Fatal("missing request id")
	}
	var ready map[string]any
	if err := json.NewDecoder(res.Body).Decode(&ready); err != nil {
		t.Fatal(err)
	}
	if ready["database"] != "memory" || ready["kubernetes"] != "disconnected" || ready["demoResetEnabled"] != false {
		t.Fatalf("ready %#v", ready)
	}
	reset, err := http.Post(srv.URL+"/api/v1/demo/reset", "application/json", bytes.NewBufferString(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	defer reset.Body.Close()
	if reset.StatusCode != http.StatusForbidden {
		t.Fatalf("reset status %d", reset.StatusCode)
	}

	statusRes, err := http.Get(srv.URL + "/api/v1/kubernetes/status")
	if err != nil {
		t.Fatal(err)
	}
	defer statusRes.Body.Close()
	var status map[string]any
	if err := json.NewDecoder(statusRes.Body).Decode(&status); err != nil {
		t.Fatal(err)
	}
	if status["connectivity"] != "disconnected" || status["source"] != "none" {
		t.Fatalf("k8s status %#v", status)
	}
}
