package telemetry

import (
	"context"
	"net/url"
	"strings"
	"testing"
)

func TestBuildQueriesRejectsUnsafeNames(t *testing.T) {
	if _, _, _, _, _, err := BuildQueries(`payment-api",namespace="x`, "demo-shop", "1m"); err == nil {
		t.Fatal("expected rejection")
	}
	if _, _, _, _, _, err := BuildQueries("payment-api", "demo-shop", "30d"); err == nil {
		t.Fatal("expected window rejection")
	}
	requests, errors, _, p95, _, err := BuildQueries("payment-api", "demo-shop", "1m")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(requests, `service="payment-api"`) || !strings.Contains(errors, `status_code=~"5.."`) {
		t.Fatalf("queries %s %s", requests, errors)
	}
	if !strings.Contains(p95, "histogram_quantile(0.95") {
		t.Fatal(p95)
	}
}

func TestParseVector(t *testing.T) {
	value, ok, err := ParseVector([]byte(`{"status":"success","data":{"resultType":"vector","result":[{"value":[1,"12.5"]}]}}`))
	if err != nil || !ok || value != 12.5 {
		t.Fatalf("%v %v %v", value, ok, err)
	}
	_, ok, err = ParseVector([]byte(`{"status":"success","data":{"resultType":"vector","result":[{"value":[1,"NaN"]}]}}`))
	if err != nil || ok {
		t.Fatalf("nan ok=%v err=%v", ok, err)
	}
	_, ok, err = ParseVector([]byte(`{"status":"success","data":{"resultType":"vector","result":[]}}`))
	if err != nil || ok {
		t.Fatalf("empty ok=%v err=%v", ok, err)
	}
}

type scripted struct {
	bodies map[string]string
}

func (s scripted) Get(_ context.Context, path string, query url.Values) (int, []byte, error) {
	if path == "/-/ready" {
		return 200, []byte("ok"), nil
	}
	body := s.bodies[query.Get("query")]
	if body == "" {
		body = `{"status":"success","data":{"resultType":"vector","result":[]}}`
	}
	return 200, []byte(body), nil
}

func TestServiceWindowMath(t *testing.T) {
	requests, errors, _, p95, _, err := BuildQueries("payment-api", "demo-shop", "1m")
	if err != nil {
		t.Fatal(err)
	}
	client := Prometheus{Get: scripted{bodies: map[string]string{
		requests: `{"status":"success","data":{"resultType":"vector","result":[{"value":[1,"100"]}]}}`,
		errors:   `{"status":"success","data":{"resultType":"vector","result":[{"value":[1,"10"]}]}}`,
		p95:      `{"status":"success","data":{"resultType":"vector","result":[{"value":[1,"0.42"]}]}}`,
	}}}
	snap, err := client.ServiceWindow(context.Background(), "payment-api", "demo-shop", "1m")
	if err != nil {
		t.Fatal(err)
	}
	if !snap.Available || snap.ErrorRate < 0.099 || snap.ErrorRate > 0.101 || snap.P95 != 0.42 {
		t.Fatalf("%#v", snap)
	}
	if snap.RequestRate < 1.6 || snap.RequestRate > 1.7 {
		t.Fatalf("rate %v", snap.RequestRate)
	}
}

func TestUnavailableWhenGetterMissing(t *testing.T) {
	snap, err := (Prometheus{}).ServiceWindow(context.Background(), "payment-api", "demo-shop", "1m")
	if err != nil || snap.Available || snap.Source != "unavailable" {
		t.Fatalf("%v %#v", err, snap)
	}
}
