package rollout

import "github.com/opspilot/opspilot/apps/api/internal/telemetry"

// Evaluate is the SLO gate. Missing latency or too few requests is not a pass.
func Evaluate(sample telemetry.Snapshot, ready bool) Analysis {
	out := Analysis{
		Result: ResultInsufficient, Requests: sample.Requests, Errors: sample.Errors,
		Error: sample.ErrorRate, P95: sample.P95, Ready: ready,
	}
	if !sample.Available || sample.Requests < MinRequests || !sample.LatencyKnown || !ready {
		return out
	}
	if sample.ErrorRate > MaxError || sample.P95 > MaxP95 {
		out.Result = ResultFail
		return out
	}
	out.Result = ResultPass
	return out
}
