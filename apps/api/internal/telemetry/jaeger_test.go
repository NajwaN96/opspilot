package telemetry

import (
	"context"
	"testing"
)

func TestParseTraceTreeAndStripSecrets(t *testing.T) {
	body := []byte(`{
		"data": [{
			"traceID": "0123456789abcdef0123456789abcdef",
			"processes": {"p1": {"serviceName": "storefront"}, "p2": {"serviceName": "checkout-api"}, "p3": {"serviceName": "payment-api"}},
			"spans": [
				{"spanID": "aa", "operationName": "GET /buy", "startTime": 1000000, "duration": 800000, "processID": "p1", "tags": [{"key":"http.status_code","value":500},{"key":"http.request.header.authorization","value":"secret"}]},
				{"spanID": "bb", "operationName": "GET /checkout", "references": [{"refType":"CHILD_OF","spanID":"aa"}], "startTime": 1100000, "duration": 700000, "processID": "p2", "tags": [{"key":"http.status_code","value":500}]},
				{"spanID": "cc", "operationName": "GET /pay", "references": [{"refType":"CHILD_OF","spanID":"bb"}], "startTime": 1200000, "duration": 500000, "processID": "p3", "tags": [{"key":"error","value":true},{"key":"db.system","value":"postgresql"},{"key":"http.request.header.cookie","value":"session"}]}
			]
		}]
	}`)
	traces, err := ParseTraces(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(traces) != 1 || traces[0].RootService != "storefront" || traces[0].Status != "error" || traces[0].Spans != 3 {
		t.Fatalf("%#v", traces[0].TraceSummary)
	}
	if len(traces[0].Tree) != 3 || traces[0].Tree[0].Service != "storefront" || traces[0].Tree[2].Depth != 2 {
		t.Fatalf("%#v", traces[0].Tree)
	}
	if traces[0].Tree[2].Service != "payment-api" || traces[0].Tree[2].Attributes["db.system"] != "postgresql" {
		t.Fatalf("%#v", traces[0].Tree[2])
	}
	for _, span := range traces[0].Tree {
		for key, value := range span.Attributes {
			if key != "http.status_code" && key != "db.system" && key != "slow" {
				t.Fatalf("unexpected attribute %s=%s", key, value)
			}
			if value == "secret" || value == "session" {
				t.Fatal("sensitive value leaked")
			}
		}
	}
}

func TestTraceIDRule(t *testing.T) {
	if _, err := (Jaeger{}).Trace(context.Background(), "not a trace"); err == nil {
		t.Fatal("expected rejection")
	}
}
