# Demo script

About eight minutes. Say what is live before you click.

## Verified End-to-End Reliability Test

This section records a manual run on the local cluster `opspilot-dev`. The numbers are the Prometheus window stored on the incident. They are not the seeded `INC-142` simulation. `INC-142` stays a separate simulated investigation and was not used as evidence for this run.

The owner performed the steps in the browser. The AI Investigator returned unauthorized. Deterministic diagnosis and the rollback still completed.

1. **Healthy baseline.** `demo-shop/payment-api` was `opspilot-demo:1.4.2`, Ready 1/1. `payment-api-canary` desired 0.
2. **Bad release.** Reliability Lab, **Deploy Bad payment**. The Deployment became `opspilot-demo:1.5.0-bad` and stayed Ready.
3. **Real degradation.** The traffic generator kept calling payment-api. The bad image answers about 30% of `GET /pay` with HTTP 500 and adds latency. The pod was not crashed.
4. **Prometheus detection.** Rule `PAYMENT_API_RELIABILITY_DEGRADATION` read a one-minute window: 77 requests, 34.6% error rate, p95 975 ms. The rule needs at least 20 requests and either an error rate above 5% or p95 above 300 ms, twice in a row.
5. **Automatic incident creation.** The detector opened `INC-REAL-ac2c87e5` at 2026-09-30 11:31:13 UTC. The button does not insert the incident. A second open incident with the same fingerprint is refused while this one is active.
6. **Evidence correlation.** The incident stored the Prometheus window, the workload version `1.5.0-bad`, Ready 1/1, and the trace list from Jaeger.
7. **Deterministic diagnosis.** The cause text is the known bad release, not an AI narrative. Ready replicas do not explain the errors. The OpenAI investigator was unauthorized and did not choose the action.
8. **Human approval.** The proposal was `rollback-payment-api` from `1.5.0-bad` to `1.4.2`. The owner clicked **Approve & Execute**.
9. **Constrained rollback.** The executor changed only `demo-shop/payment-api` to `opspilot-demo:1.4.2`.
10. **Recovery verification.** Prometheus had to show a healthy window after the new pod was Ready. The verifier does not trust the rollout by itself.
11. **Incident resolution.** `INC-REAL-ac2c87e5` is `resolved` (resolved at 2026-09-30 11:51:29 UTC). Telemetry recovery is recorded. The row is kept.

After the run the lab was returned to `opspilot-demo:1.4.2`, Ready 1/1, canary desired 0, with no active fault, remediation, or rollout. The public Lambda site does not show this incident. It serves sanitized samples and refuses POST.

## Open

OpsPilot is a Kubernetes reliability control plane. The laptop cluster is live. The public URL is a read-only AWS portfolio. The EKS stack is plan only and has not been applied.

## Architecture

Open Architecture. Walk the three columns: Lambda and OIDC, k3d through approval to verification, OpenTofu marked plan only.

## Catalog

Open Developer Portal. Show payment-api: owner, canary strategy, SLO, and the scorecard. Security context is MISSING on the running demo and present on the golden-path template. That is intentional.

Open Golden path. Generate a preview for a new Go service. On AWS, stop at the preview. Locally, the write goes to `generated/services` and will not replace payment-api.

## SLO and a bad release

On the local lab, show payment-api healthy at `opspilot-demo:1.4.2`. From Reliability Lab, deploy the controlled bad release. Wait for Prometheus to move the error rate. An incident opens from the detector, not from the button alone.

## Incident

Open the incident. The lifecycle runs from detected through approval to resolved. Evidence shows the Prometheus window, the trace into `charge`, and the image. The AI panel either cites the snapshot or says AI Investigator Unavailable. Detection still stands. Approve the rollback. The executor changes only that Deployment. Prometheus recovers, and the audit row is on the incident.

Reset Demo returns the baseline. Do not leave the bad image running.

## Progressive delivery

Show a good canary promoting through 5, 25, 50, and 100 with the SLO gate. Show a bad canary failing the gate and aborting back to 1.4.2. The stable Deployment is the one that stays.

## CI and AWS

Open the GitHub Actions run for `main`. The gates include tests, scans, both OpenTofu stacks, and the zero-cost policy. The deploy job assumes `opspilot-github-deployer` and updates the Lambda. Settings shows the git commit. A POST to reset or remediate on the public URL returns 403.

## Close

Security page: AI read-only, public AWS read-only, remediation allowlisted, EKS plan only, no long-lived access keys.
