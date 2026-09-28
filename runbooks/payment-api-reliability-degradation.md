# Payment API reliability degradation

Rule: `PAYMENT_API_RELIABILITY_DEGRADATION`

Service: `payment-api` in namespace `demo-shop` on cluster `opspilot-dev`

## Symptoms

- OpsPilot opens `INC-REAL-…` with title “Payment API reliability degradation”.
- Source is Detection Engine. Simulation is No.
- Users of `storefront` `/buy` see HTTP 500s and responses around 500ms when the Reliability Lab fault is active.

## Signals

Prometheus, one-minute window:

- `http_requests_total{service="payment-api",namespace="demo-shop"}`
- the same counter with `status_code=~"5.."`
- `histogram_quantile(0.95, … http_request_duration_seconds_bucket …)`

Jaeger traces rooted at `storefront` with a slow or error `payment-api` span named `GET /pay` or `db.query`.

Kubernetes: desired and ready replicas for Deployment `payment-api`.

## Thresholds

| Check | Value |
| --- | --- |
| Evaluation interval | 15s |
| Window | 1m |
| Minimum requests | 20 |
| Error rate | greater than 5% |
| p95 latency | greater than 300ms |
| Sustain | 2 consecutive breaches |
| Recovery | 2 consecutive healthy evaluations |
| Missing telemetry | hold; do not open and do not recover |

The local SLO is separate: 99.9% availability over a 15-minute observation window, only after 20 requests. It is not a 30-day claim.

## Investigation

1. Confirm the incident origin is Detection Engine and the metrics source is Prometheus.
2. Read the timeline for which threshold tripped.
3. Open recent traces and see whether slow or failed spans sit on `payment-api` while `storefront` and `checkout-api` only wait.
4. Compare ready replicas with desired replicas. If they match, pod availability is not the explanation.
5. Check Reliability Lab for a running `payment-api-degraded` experiment.

## Safe operator response

If a Reliability Lab experiment is running, stop it. That clears the in-process fault. It does not roll back a Deployment and it is not production remediation.

The API still rejects action `rollback` on `INC-REAL-…`. Action `rollback-payment-api` is allowed only when the observed version is `1.5.0-bad`. See `runbooks/payment-api-rollback.md`.

The lab fault also expires on its own. The maximum duration is 5 minutes. The console offers 30, 60, or 120 seconds.

## Verification

- Prometheus error rate is at or below 5% and p95 is at or below 300ms for two evaluations, with at least 20 requests in the minute.
- The incident timeline gains “Telemetry recovered”.
- The incident stays `investigating`. It is not closed automatically.
- No second `INC-REAL-…` row appears for the same fingerprint.
- `payment-api` remains Ready.
