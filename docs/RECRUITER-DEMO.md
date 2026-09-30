# Recruiter demo

Five minutes. Open with this sentence:

> OpsPilot is a Kubernetes reliability control plane that detects service degradation, correlates telemetry, proposes constrained remediation, requires human approval, executes an allowlisted Kubernetes rollback, and verifies recovery.

Say which screen is which venue before you click. **LOCAL — LIVE** is the laptop cluster. **AWS PUBLIC PORTFOLIO — READ ONLY** is the Lambda URL and is not attached to that cluster. **AWS — PLAN ONLY** is the unapplied EKS design.

## Walk

1. **Architecture.** Three columns: the public Lambda and GitHub OIDC, the local path from detection to verification, and OpenTofu marked plan only.
2. **Developer portal.** Catalog, golden-path preview, and the scorecard. Missing security context on the running demo is reported as missing. It is not given a numeric score.
3. **Service catalog.** `payment-api` owner, Go runtime, canary strategy, SLO, and runbook link.
4. **SLOs.** Local payment-api availability is computed from Prometheus. The public page says the numbers there are a sanitized sample.
5. **Observability.** Prometheus, the OpenTelemetry Collector, Jaeger, Grafana, and Alertmanager are in the local cluster. The console does not accept PromQL from the browser.
6. **Bad release.** On the local Reliability Lab only, **Deploy Bad payment** sets `demo-shop/payment-api` to `opspilot-demo:1.5.0-bad`. The pod stays Ready.
7. **Incident detection.** Rule `PAYMENT_API_RELIABILITY_DEGRADATION` opens `INC-REAL-…` after two bad one-minute evaluations with at least 20 requests. The verified run was `INC-REAL-ac2c87e5`: 77 requests, 34.6% error rate, p95 975 ms.
8. **Evidence.** The incident shows that Prometheus window, Jaeger traces, and the image.
9. **Diagnosis.** The cause is the known bad release. In the verified run the AI Investigator was unauthorized. Detection and the proposal did not depend on it.
10. **Human approval.** **Approve & Execute** is required. The proposal is `rollback-payment-api` from `1.5.0-bad` to `1.4.2`.
11. **Rollback.** The executor updates only that Deployment.
12. **Recovery.** Prometheus must look healthy again. Then the incident is `resolved`. `INC-REAL-ac2c87e5` ended resolved on `1.4.2`.
13. **Progressive delivery.** A healthy canary (`1.5.0`) moves 5%, 25%, 50%, then promote. A bad canary (`1.6.0-bad`) fails the same gate and aborts. Stable `1.4.2` is a different Deployment from the canary.
14. **GitHub Actions and OIDC.** Push to `main` runs the test, scan, OpenTofu, and zero-cost gates, then assumes `opspilot-github-deployer` and updates the existing Lambda. There is no long-lived AWS access key in the workflow.
15. **AWS portfolio.** The public URL is read-only. POST returns 403.
16. **Security boundaries.** AI is read-only. Discovery is read-only. Remediation is one allowlisted action. EKS is plan only.

`INC-142` is a seeded simulation on the same Incidents page. Do not present its 12.4% error rate or v1.8.2 story as the live test.

## What is REAL

- Local k3d cluster `opspilot-dev`
- `payment-api` and the rest of demo-shop
- The traffic generator
- Prometheus metrics
- OpenTelemetry traces in Jaeger
- Incident detection from those metrics
- PostgreSQL persistence of incidents, approvals, and the audit trail
- Human approval
- The constrained Kubernetes rollback of `payment-api`
- Recovery verification against Prometheus
- Progressive delivery of the payment-api canary

## What is DEMO / PLAN ONLY

**Public demo.** The AWS Lambda function URL is a portfolio presentation. It serves the console and a sanitized API. It does not read the local cluster.

**Plan only.** The AWS EKS architecture under `infra/terraform`. CI runs `tofu validate`. The stack is not applied.

The verified local result is written up in [DEMO.md](DEMO.md#verified-end-to-end-reliability-test).
