package detection

import "fmt"

// Finding is a deterministic explanation. It has no confidence percentage.
type Finding struct {
	LikelyCause   string
	Supporting    []string
	Contradicting []string
}

type Facts struct {
	ErrorRate    float64 `json:"errorRate"`
	P95          float64 `json:"p95"`
	Requests     float64 `json:"requests"`
	Ready        int     `json:"ready"`
	Desired      int     `json:"desired"`
	Restarts     int     `json:"restarts"`
	SlowOrFailed int     `json:"slowOrFailed"`
	Version      string  `json:"version"`
}

func Diagnose(facts Facts) Finding {
	var supporting []string
	var contradicting []string
	if facts.Requests >= MinRequests && facts.ErrorRate > ErrorRate {
		supporting = append(supporting, fmt.Sprintf("payment-api error rate %.1f%% exceeded the %.0f%% threshold over the local window", facts.ErrorRate*100, ErrorRate*100))
	}
	if facts.Requests >= MinRequests && facts.P95 > P95Seconds {
		supporting = append(supporting, fmt.Sprintf("payment-api p95 latency %.0fms exceeded the %.0fms threshold", facts.P95*1000, P95Seconds*1000))
	}
	if facts.SlowOrFailed > 0 {
		supporting = append(supporting, fmt.Sprintf("%d recent slow or failed traces include payment-api", facts.SlowOrFailed))
	}
	if facts.Version != "" {
		supporting = append(supporting, "current workload version is "+facts.Version)
	}
	if facts.Desired > 0 && facts.Ready >= facts.Desired {
		contradicting = append(contradicting, "Kubernetes replicas remained ready, which does not support a pod-availability failure")
	}
	if facts.Desired > 0 && facts.Ready < facts.Desired {
		supporting = append(supporting, fmt.Sprintf("only %d of %d replicas are ready", facts.Ready, facts.Desired))
	}
	if facts.Restarts > 0 {
		supporting = append(supporting, fmt.Sprintf("pods restarted %d times", facts.Restarts))
	}
	cause := "Signals are mixed. Compare the error rate, latency, traces, and replica counts before treating this as an application fault."
	if facts.Desired > 0 && facts.Ready >= facts.Desired && (facts.ErrorRate > ErrorRate || facts.P95 > P95Seconds) {
		cause = "Application-level degradation in payment-api rather than pod availability failure."
	}
	if facts.Desired > 0 && facts.Ready < facts.Desired {
		cause = "payment-api has unavailable replicas. An application fault is not the only explanation."
	}
	return Finding{LikelyCause: cause, Supporting: supporting, Contradicting: contradicting}
}
