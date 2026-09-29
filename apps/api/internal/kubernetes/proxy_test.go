package kubernetes

import (
	"net/http"
	"testing"
)

func TestProxyAllowlist(t *testing.T) {
	allowed := []struct{ ns, svc, method, path string }{
		{"opspilot-system", "prometheus", http.MethodGet, "/api/v1/query"},
		{"opspilot-system", "prometheus", http.MethodGet, "/-/ready"},
		{"opspilot-system", "jaeger", http.MethodGet, "/api/traces"},
		{"opspilot-system", "jaeger", http.MethodGet, "/api/traces/abc123"},
		{"opspilot-system", "otel-collector", http.MethodGet, "/"},
		{"demo-shop", "payment-api", http.MethodPost, "/internal/fault"},
		{"demo-shop", "checkout-api", http.MethodPost, "/internal/canary"},
		{"opspilot-system", "alertmanager", http.MethodGet, "/-/ready"},
		{"opspilot-system", "grafana", http.MethodGet, "/api/health"},
		{"opspilot-system", "prometheus", http.MethodGet, "/api/v1/rules"},
	}
	for _, item := range allowed {
		if !proxyAllowed(item.ns, item.svc, item.method, item.path) {
			t.Fatalf("expected allow %#v", item)
		}
	}
	denied := []struct{ ns, svc, method, path string }{
		{"demo-shop", "payment-api", http.MethodGet, "/internal/fault"},
		{"demo-shop", "storefront", http.MethodPost, "/internal/fault"},
		{"opspilot-system", "prometheus", http.MethodPost, "/api/v1/query"},
		{"opspilot-system", "prometheus", http.MethodGet, "/api/v1/admin/tsdb/delete_series"},
		{"kube-system", "kube-apiserver", http.MethodGet, "/"},
		{"default", "postgres", http.MethodPost, "/internal/fault"},
		{"demo-shop", "payment-api", http.MethodPost, "/bin/sh"},
		{"opspilot-system", "jaeger", http.MethodGet, "/api/traces/../../secret"},
		{"demo-shop", "payment-api", http.MethodPost, "/internal/fault/extra"},
		{"demo-shop", "checkout-api", http.MethodGet, "/internal/canary"},
		{"demo-shop", "checkout-api", http.MethodPost, "/internal/canary/extra"},
		{"opspilot-system", "alertmanager", http.MethodPost, "/api/v2/silences"},
		{"opspilot-system", "grafana", http.MethodGet, "/api/datasources"},
	}
	for _, item := range denied {
		if proxyAllowed(item.ns, item.svc, item.method, item.path) {
			t.Fatalf("expected deny %#v", item)
		}
	}
}
