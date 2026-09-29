package investigate

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestOpenAITimeoutDoesNotBecomeAFixture(t *testing.T) {
	provider := OpenAI{
		ModelName: "gpt-4.1-mini",
		APIKey:    "test-key",
		Client: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.Header.Get("Authorization") == "" {
				t.Fatal("missing auth")
			}
			if strings.Contains(r.Header.Get("Authorization"), "logged") {
				t.Fatal("key logged")
			}
			return nil, context.DeadlineExceeded
		}), Timeout: time.Millisecond},
	}
	_, _, err := provider.Investigate(context.Background(), Snapshot{IncidentID: "INC"})
	if err == nil || provider.Name() != "openai" || !provider.Real() {
		t.Fatalf("err=%v", err)
	}
	if errors.Is(err, io.EOF) {
		t.Fatal(err)
	}
}

func TestQuotaIsNotRelabeledAsFixture(t *testing.T) {
	provider := OpenAI{
		APIKey: "test-key",
		Client: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusTooManyRequests,
				Body:       io.NopCloser(strings.NewReader(`{"error":{"type":"insufficient_quota","code":"credit_balance_exhausted","message":"no credits"}}`)),
				Header:     make(http.Header),
			}, nil
		})},
	}
	_, _, err := provider.Investigate(context.Background(), Snapshot{IncidentID: "INC"})
	if err == nil || !provider.Real() || !strings.Contains(err.Error(), "quota") || strings.Contains(err.Error(), "no credits") {
		t.Fatalf("%v", err)
	}
}

func TestOpenAIDoesNotAcceptMalformedOutput(t *testing.T) {
	provider := OpenAI{
		APIKey: "test-key",
		Client: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(`{"choices":[{"message":{"content":"not-json"}}]}`)),
				Header:     make(http.Header),
			}, nil
		})},
	}
	_, _, err := provider.Investigate(context.Background(), Snapshot{})
	if err == nil || !strings.Contains(err.Error(), "invalid") {
		t.Fatalf("%v", err)
	}
}
