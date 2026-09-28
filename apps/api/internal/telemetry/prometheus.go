package telemetry

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"regexp"
	"strconv"
	"time"
)

var namePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,62}$`)

// Getter reads one local HTTP path. Implementations must not accept PromQL from the browser.
type Getter interface {
	Get(ctx context.Context, path string, query url.Values) (int, []byte, error)
}

type Snapshot struct {
	Available    bool
	Source       string
	Message      string
	Service      string
	Namespace    string
	Updated      time.Time
	Requests     float64
	Errors       float64
	ErrorRate    float64
	P50          float64
	P95          float64
	P99          float64
	Availability float64
	RequestRate  float64
}

type Prometheus struct {
	Get Getter
	Now func() time.Time
}

func (p Prometheus) now() time.Time {
	if p.Now != nil {
		return p.Now()
	}
	return time.Now()
}

// ServiceWindow queries counters and a latency histogram for one known service.
// window is 1m or 15m. The PromQL is built here.
func (p Prometheus) ServiceWindow(ctx context.Context, service, namespace, window string) (Snapshot, error) {
	out := Snapshot{Source: "unavailable", Service: service, Namespace: namespace, Message: "Telemetry unavailable", Updated: p.now()}
	if p.Get == nil {
		return out, nil
	}
	requestsQ, errorsQ, p50Q, p95Q, p99Q, err := BuildQueries(service, namespace, window)
	if err != nil {
		out.Message = err.Error()
		return out, nil
	}
	seconds := 60.0
	if window == "15m" {
		seconds = 15 * 60
	}
	requests, ok, err := p.instant(ctx, requestsQ)
	if err != nil || !ok {
		return out, nil
	}
	errors, _, err := p.instant(ctx, errorsQ)
	if err != nil {
		return out, nil
	}
	p50, _, _ := p.instant(ctx, p50Q)
	p95, _, _ := p.instant(ctx, p95Q)
	p99, _, _ := p.instant(ctx, p99Q)
	errorRate := 0.0
	availability := 1.0
	if requests > 0 {
		errorRate = errors / requests
		availability = 1 - errorRate
	}
	return Snapshot{
		Available:    true,
		Source:       "prometheus",
		Service:      service,
		Namespace:    namespace,
		Updated:      p.now(),
		Requests:     requests,
		Errors:       errors,
		ErrorRate:    errorRate,
		P50:          p50,
		P95:          p95,
		P99:          p99,
		Availability: availability,
		RequestRate:  requests / seconds,
	}, nil
}

func (p Prometheus) Ready(ctx context.Context) error {
	if p.Get == nil {
		return fmt.Errorf("prometheus is not configured")
	}
	code, _, err := p.Get.Get(ctx, "/-/ready", nil)
	if err != nil {
		return err
	}
	if code != 200 {
		return fmt.Errorf("prometheus ready %d", code)
	}
	return nil
}

func (p Prometheus) instant(ctx context.Context, query string) (float64, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	code, body, err := p.Get.Get(ctx, "/api/v1/query", url.Values{"query": []string{query}})
	if err != nil {
		return 0, false, err
	}
	if code != 200 {
		return 0, false, fmt.Errorf("prometheus query status %d", code)
	}
	return ParseVector(body)
}

// BuildQueries returns the only PromQL the control plane will send for a service.
func BuildQueries(service, namespace, window string) (requests, errors, p50, p95, p99 string, err error) {
	if !namePattern.MatchString(service) || !namePattern.MatchString(namespace) {
		return "", "", "", "", "", fmt.Errorf("service identity is not allowed")
	}
	if window != "1m" && window != "15m" {
		return "", "", "", "", "", fmt.Errorf("window is not allowed")
	}
	selector := fmt.Sprintf(`service="%s",namespace="%s"`, service, namespace)
	requests = fmt.Sprintf(`sum(increase(http_requests_total{%s}[%s]))`, selector, window)
	errors = fmt.Sprintf(`sum(increase(http_requests_total{%s,status_code=~"5.."}[%s]))`, selector, window)
	base := fmt.Sprintf(`sum by (le) (rate(http_request_duration_seconds_bucket{%s}[%s]))`, selector, window)
	p50 = fmt.Sprintf(`histogram_quantile(0.50, %s)`, base)
	p95 = fmt.Sprintf(`histogram_quantile(0.95, %s)`, base)
	p99 = fmt.Sprintf(`histogram_quantile(0.99, %s)`, base)
	return requests, errors, p50, p95, p99, nil
}

func ParseVector(body []byte) (float64, bool, error) {
	var payload struct {
		Status string `json:"status"`
		Data   struct {
			ResultType string `json:"resultType"`
			Result     []struct {
				Value []any `json:"value"`
			} `json:"result"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return 0, false, err
	}
	if payload.Status != "success" || payload.Data.ResultType != "vector" {
		return 0, false, fmt.Errorf("unexpected prometheus payload")
	}
	if len(payload.Data.Result) == 0 || len(payload.Data.Result[0].Value) < 2 {
		return 0, false, nil
	}
	raw, ok := payload.Data.Result[0].Value[1].(string)
	if !ok {
		return 0, false, fmt.Errorf("prometheus value is not a string")
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, false, nil
	}
	return value, true, nil
}
