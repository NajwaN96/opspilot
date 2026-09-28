# OpsPilot

Kubernetes reliability control plane.

Production incidents are still reconstructed by hand: a deploy lands, latency moves, someone pastes logs into a channel, and a rollback waits on a person who has standing cluster-admin access. OpsPilot is the control plane for that loop. It is meant to detect a change, correlate it with telemetry, explain the likely cause, propose one constrained action, require a human when the policy says so, execute only that action, and verify recovery.

This repository is a local control plane. PostgreSQL stores operational state, a read-only client discovers workloads on a local Kubernetes cluster, and demo-shop emits real Prometheus metrics and OpenTelemetry traces. A deterministic rule can open an incident from that telemetry. `INC-142` still uses a simulated rollback. A detected incident can approve one real rollback: `demo-shop/payment-api` from `1.5.0-bad` to `1.4.2`. Nothing here talks to a cloud account or a model.

## What it is

OpsPilot is an internal operations console plus a small Go API.

The console is the place an on-call engineer would work an incident: service health, a timeline, evidence, a probable cause, and a single remediation with an explicit risk and policy decision. The API keeps those reads and the approval behind repository and executor interfaces so a later PostgreSQL store or a narrow Kubernetes client can replace the simulation without rewriting the UI.

## Incident lifecycle

The product loop, end to end:

1. **Detect** a symptom against an SLO or a deploy.
2. **Correlate** it with the rollout, logs, traces, and cluster events.
3. **Diagnose** a probable cause and show the evidence.
4. **Recommend** one action, with a risk class.
5. **Approve** when policy requires a person.
6. **Remediate** through a constrained executor, not a general admin client.
7. **Verify** that the symptom returned to baseline.
8. **Learn** from the closed incident. Not built yet.

`INC-142` still walks that loop on seeded telemetry. A detected `payment-api` incident can walk it on live Prometheus, traces, and the local cluster: the rule recommends a rollback only for release `1.5.0-bad`, a person approves it, and the executor changes that one Deployment. Step 8 is still planned.

## Safety model

An investigator, including a future model, must never hold unrestricted Kubernetes administrative access.

The intended path is:

```
recommendation → deterministic policy check → risk class → human approval when required → constrained executor → verification
```

The simulated executor still only logs the `INC-142` request. The live executor accepts one already-authorized request, `rollback-payment-api`, and updates only `demo-shop/payment-api` from image `opspilot-demo:1.5.0-bad` to `opspilot-demo:1.4.2`. It does not decide policy, and it rejects every other namespace, Deployment, and image. There is no generic apply, patch, or exec endpoint.

## Architecture

```
                        ┌──────────────┐
                        │   Next.js    │
                        │   Console    │
                        └──────┬───────┘
                               │
                               ▼
                        ┌──────────────┐
                        │ OpsPilot API │
                        │      Go      │
                        └──────┬───────┘
                               │
           ┌───────────────────┼───────────────────┐
           │                   │                   │
           ▼                   ▼                   ▼
     PostgreSQL           Kubernetes          Prometheus
                                                │
demo-shop                                       │
    ├── Metrics ────────────────────────────────┘
    └── OTLP → OpenTelemetry Collector → Jaeger → OpsPilot
```

Future collection and action path:

```
Kubernetes
  → OpenTelemetry
  → Prometheus / logs / traces
  → OpsPilot
  → incident engine
  → investigator
  → remediation proposal
  → policy engine
  → human approval
  → constrained Kubernetes action
  → recovery verification
```

See [docs/architecture/overview.md](docs/architecture/overview.md).

## Status

### Real now

- PostgreSQL persistence for clusters, services, incidents, timeline events, approvals, executions, and the audit trail
- Local Kubernetes cluster `opspilot-dev` (k3d) with namespaces `opspilot-system` and `demo-shop`
- Instrumented demo-shop services (`storefront` → `checkout-api` → `payment-api`, plus orders and inventory) and a traffic generator
- Prometheus, the OpenTelemetry Collector, and Jaeger in `opspilot-system`
- Read-only discovery of Deployments, Pods, Services, and Events in `demo-shop`
- Server-built Prometheus queries and Jaeger trace reads for discovered services
- Rule `PAYMENT_API_RELIABILITY_DEGRADATION`, which opens one `INC-REAL-…` incident from live telemetry
- A constrained Reliability Lab fault on `payment-api` only: 40% HTTP 500 and 500ms latency, 30/60/120 seconds, auto-expire at 5 minutes
- A development-only bad release, `payment-api` `1.5.0-bad`, and a human-approved rollback to `1.4.2` verified with Prometheus
- A local 15-minute availability SLO for `payment-api` (target 99.9%)
- `GET /health` and `GET /ready` (PostgreSQL, Kubernetes, Prometheus, collector, Jaeger)
- Development-only `POST /api/v1/demo/reset`, which restores `INC-142` and does not delete cluster data

### Simulated

- `production-01` incident telemetry, logs, traces, and the events embedded in `INC-142`
- The probable cause and its confidence on `INC-142`
- The rollback executor and the health recovery that follows approval of `INC-142`
- The original Reliability Lab scenarios that do not touch the cluster

### Planned

- A learning loop
- GitOps and AWS/EKS
- Any remediation other than `demo-shop/payment-api` `1.5.0-bad` → `1.4.2`

AWS, Terraform, Grafana, Loki, Argo, and an AI provider are intentionally not integrated. The Kubernetes reader still cannot create, update, delete, or exec. The payment rollback is a separate method, and the detection engine is not given it. Generic actions such as `rollback` are still rejected for `INC-REAL-…` incidents.

## Current MVP

Cluster `production-01` is degraded because `payment-api` is in `INC-142`.

| | |
| --- | --- |
| Severity | SEV-2 |
| Status | Investigating, until you approve the rollback |
| Service | payment-api |
| Availability | 98.71% |
| p95 latency | 1.8s |
| Error rate | 12.4% |
| Version | v1.8.2, previous v1.8.1 |
| Probable cause | Database connection leak introduced by payment-api:v1.8.2 |
| Confidence | 91% (simulated) |
| Recommendation | Roll back to v1.8.1. Risk LOW. Policy Allowed. |

Open the incident and choose **Approve & Execute**. The UI walks approval, rollback, rollout, verification, and resolution. Error rate, latency, and database connections move to the recovered values. Those numbers are simulated. The approval is stored in PostgreSQL and remains after an API restart. **Reset Demo**, shown only when the API is not in production mode, restores `INC-142` without deleting Kubernetes rows.

## Running locally

Requirements: Node.js 22, npm, Go 1.22, Docker, k3d, kubectl.

PostgreSQL:

```bash
docker compose -f infra/docker-compose.yml up -d postgres
```

Local cluster, demo workloads, and observability:

```bash
docker build -t opspilot-demo:0.3.0 apps/demo
infra/scripts/up-dev-cluster.sh
```

The script creates `opspilot-dev` if needed, points kubeconfig at the node IP when the k3d load balancer cannot reach the API server, imports local images, and applies `demo-shop` plus Prometheus, the collector, and Jaeger. In-cluster image pulls often time out in this environment, so images are imported with `ctr`.

Terminal one:

```bash
cd apps/api
export OPSPILOT_DATABASE_URL=postgres://opspilot:opspilot@127.0.0.1:5432/opspilot?sslmode=disable
export OPSPILOT_ENV=development
go run ./cmd/opspilot
```

The API listens on [http://127.0.0.1:8094](http://127.0.0.1:8094). Migrations run on startup. `GET /health` is liveness. `GET /ready` reports database and Kubernetes connectivity. Without `OPSPILOT_DATABASE_URL` the API keeps the in-memory simulation and does not persist approvals.

Terminal two:

```bash
cd apps/web
npm install
npm run dev
```

The console listens on [http://127.0.0.1:3461](http://127.0.0.1:3461) and proxies `/api`, `/health`, and `/ready` to the API. No account is required. Kubernetes discovery uses the local kubeconfig and only lists `demo-shop`.

Useful environment variables are listed in [.env.example](.env.example). None are required for the defaults above.

Checks:

```bash
cd apps/api && go test ./... && go build -o /tmp/opspilot ./cmd/opspilot
cd apps/web && npm test && npm run lint && npm run typecheck && npm run build
```

Docker Compose is optional and documented in [infra/README.md](infra/README.md). The supported local path is the two processes above.

## Screenshots

From the local simulated MVP.

![production-01 overview with INC-142 open](docs/images/overview.webp)

![INC-142 before approval](docs/images/incident.webp)

![INC-142 after the simulated rollback](docs/images/incident-resolved.webp)

## Layout

```
apps/web     Next.js console
apps/api     Go control plane
apps/demo    Instrumented demo-shop services
docs         Architecture and ADRs
demo         Walkthrough for INC-142
infra        Cluster, Prometheus, collector, Jaeger
platform     Executor boundary notes
runbooks     payment-api rollback and reliability degradation
```

## Roadmap

1. Keep the investigator off the mutating client. Stopping a Reliability Lab fault is not the payment-api rollback.
2. Add further named actions only with the same closed target list. Do not add a generic Kubernetes write API.

## Decisions

- [0001 — Monorepo](docs/adr/0001-monorepo.md)
- [0002 — Go control plane](docs/adr/0002-go-control-plane.md)
- [0003 — Safe remediation](docs/adr/0003-safe-remediation-model.md)
- [0004 — PostgreSQL persistence](docs/adr/0004-postgresql-persistence.md)
- [0005 — Read-only Kubernetes](docs/adr/0005-kubernetes-readonly.md)
- [0006 — Data provenance](docs/adr/0006-data-provenance.md)
- [0007 — OpenTelemetry architecture](docs/adr/0007-opentelemetry-architecture.md)
- [0008 — Prometheus query boundary](docs/adr/0008-prometheus-query-boundary.md)
- [0009 — Controlled failure injection](docs/adr/0009-controlled-failure-injection.md)
- [0010 — Deterministic incident detection](docs/adr/0010-deterministic-incident-detection.md)
- [0011 — Constrained payment-api rollback](docs/adr/0011-payment-api-rollback.md)
