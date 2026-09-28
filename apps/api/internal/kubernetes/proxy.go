package kubernetes

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// proxyAllowed is the only set of in-cluster HTTP calls the control plane may make.
// It is not a general proxy and it is not reachable from the browser.
func proxyAllowed(namespace, service, method, path string) bool {
	if strings.Contains(path, "..") || strings.Contains(path, "//") || !strings.HasPrefix(path, "/") {
		return false
	}
	switch {
	case namespace == "opspilot-system" && service == "prometheus" && method == http.MethodGet:
		return path == "/api/v1/query" || path == "/-/ready" || path == "/-/healthy"
	case namespace == "opspilot-system" && service == "jaeger" && method == http.MethodGet:
		return path == "/api/services" || path == "/api/traces" || strings.HasPrefix(path, "/api/traces/")
	case namespace == "opspilot-system" && service == "otel-collector" && method == http.MethodGet:
		return path == "/"
	case namespace == "demo-shop" && service == "payment-api" && method == http.MethodPost:
		return path == "/internal/fault"
	default:
		return false
	}
}

// Proxy performs one allow-listed request through the API server service proxy.
func (c *Client) Proxy(ctx context.Context, namespace, service string, port int, method, path string, query url.Values, body []byte, headers map[string]string) (int, []byte, error) {
	if c == nil || c.clientset == nil {
		return 0, nil, fmt.Errorf("kubernetes client is not configured")
	}
	if !proxyAllowed(namespace, service, method, path) {
		return 0, nil, fmt.Errorf("proxy target is not allowed")
	}
	req := c.clientset.CoreV1().RESTClient().Verb(method).
		Namespace(namespace).
		Resource("services").
		Name(fmt.Sprintf("%s:%d", service, port)).
		SubResource("proxy")
	if path != "/" {
		req = req.Suffix(strings.Split(strings.Trim(path, "/"), "/")...)
	}
	for key, values := range query {
		for _, value := range values {
			req = req.Param(key, value)
		}
	}
	if body != nil {
		req = req.Body(body)
	}
	for key, value := range headers {
		if key != "Content-Type" && key != "X-OpsPilot-Fault-Token" {
			return 0, nil, fmt.Errorf("proxy header is not allowed")
		}
		req = req.SetHeader(key, value)
	}
	ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	result := req.Do(ctx)
	var code int
	result.StatusCode(&code)
	raw, err := result.Raw()
	if err != nil && len(raw) == 0 {
		return code, nil, err
	}
	return code, raw, nil
}

// SetPaymentFault enables or clears the in-process payment-api fault.
// It never changes a Deployment.
func (c *Client) SetPaymentFault(ctx context.Context, token, mode string, until time.Time) error {
	if token == "" {
		return fmt.Errorf("fault token is not configured")
	}
	payload := []byte(`{"mode":"off"}`)
	if mode == "degraded" {
		payload = []byte(fmt.Sprintf(`{"mode":"degraded","until":%q}`, until.UTC().Format(time.RFC3339)))
	} else if mode != "off" {
		return fmt.Errorf("unknown fault mode")
	}
	code, raw, err := c.Proxy(ctx, "demo-shop", "payment-api", 80, http.MethodPost, "/internal/fault", nil, payload, map[string]string{
		"Content-Type":           "application/json",
		"X-OpsPilot-Fault-Token": token,
	})
	if err != nil {
		return err
	}
	if code < 200 || code >= 300 {
		return fmt.Errorf("payment-api fault endpoint returned %d: %s", code, trim(raw))
	}
	return nil
}

func trim(raw []byte) string {
	text := string(bytes.TrimSpace(raw))
	if len(text) > 180 {
		return text[:180]
	}
	return text
}

// ReadProxy is a small GET helper for telemetry backends.
func (c *Client) ReadProxy(ctx context.Context, namespace, service string, port int, path string, query url.Values) (int, []byte, error) {
	return c.Proxy(ctx, namespace, service, port, http.MethodGet, path, query, nil, nil)
}
