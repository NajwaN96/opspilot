package detection

import "time"

// Thresholds are the local-demo PAYMENT_API_RELIABILITY_DEGRADATION rule.
// A single request cannot open an incident: MinRequests must be met inside Window.
// The rule fires when volume is sufficient AND (error rate > ErrorRate OR p95 > P95).
// It must then stay true for Sustain consecutive evaluations.
// Recovery is recorded after Recover consecutive healthy evaluations.
// Missing telemetry is neither a breach nor a recovery.
const (
	RuleID       = "PAYMENT_API_RELIABILITY_DEGRADATION"
	Service      = "payment-api"
	Namespace    = "demo-shop"
	MinRequests  = 20.0
	ErrorRate    = 0.05
	P95Seconds   = 0.30
	Sustain      = 2
	Recover      = 2
	Window       = time.Minute
	SLOTarget    = 0.999
	SLOWindow    = 15 * time.Minute
	EvalInterval = 15 * time.Second
)

type Sample struct {
	OK       bool
	Requests float64
	Errors   float64
	P95      float64
	P50      float64
	P99      float64
}

type Decision struct {
	Breach  bool
	Healthy bool
	Reason  string
}

func Evaluate(sample Sample) Decision {
	if !sample.OK {
		return Decision{Reason: "telemetry unavailable"}
	}
	if sample.Requests < MinRequests {
		return Decision{Reason: "request volume below minimum"}
	}
	errorRate := 0.0
	if sample.Requests > 0 {
		errorRate = sample.Errors / sample.Requests
	}
	highError := errorRate > ErrorRate
	highLatency := sample.P95 > P95Seconds
	if highError || highLatency {
		return Decision{Breach: true, Reason: "error rate or p95 exceeded the local threshold"}
	}
	return Decision{Healthy: true, Reason: "within threshold"}
}

func Fingerprint(cluster string) string {
	return RuleID + "|" + Service + "|" + Namespace + "|" + cluster
}

// Streak advances consecutive breach and recovery counts.
// A non-decision (missing data or low volume) resets neither counter.
type Streak struct {
	Breach  int
	Healthy int
}

func (s Streak) Next(decision Decision) (Streak, string) {
	switch {
	case decision.Breach:
		s.Breach++
		s.Healthy = 0
		if s.Breach >= Sustain {
			return s, "open"
		}
		return s, "pending"
	case decision.Healthy:
		s.Healthy++
		s.Breach = 0
		if s.Healthy >= Recover {
			return s, "recovered"
		}
		return s, "pending"
	default:
		return s, "hold"
	}
}
