# OpsPilot

Kubernetes reliability control plane.

Production incidents are still reconstructed by hand: a deploy lands, latency moves, someone pastes logs into a channel, and a rollback waits on a person who has standing cluster-admin access. OpsPilot is the control plane for that loop. It is meant to detect a change, correlate it with telemetry, explain the likely cause, propose one constrained action, require a human when the policy says so, execute only that action, and verify recovery.

This repository is a local control plane. PostgreSQL stores operational state, and a read-only client discovers workloads on a local Kubernetes cluster. Incident telemetry, root-cause analysis, and rollback execution are still simulated. Nothing here talks to a cloud account, Prometheus, or a model.

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

In this MVP, steps 1–3 and 7 are seeded data. Step 4 is a fixed recommendation for `INC-142`. Steps 5–6 run for real inside the process: the API checks policy, calls a simulated executor, and advances a rollback workflow. Step 8 is planned.

## Safety model

An investigator, including a future model, must never hold unrestricted Kubernetes administrative access.

The intended path is:

```
recommendation → deterministic policy check → risk class → human approval when required → constrained executor → verification
```

The executor receives one already-authorized request (`rollback` of one Deployment from one version to another). It does not decide policy and it does not accept an arbitrary manifest. Today that executor only logs the request. `kubectl` is not invoked.

## Architecture

```
browser
  → Next.js console (apps/web)
    → Go API (apps/api)
      → service layer (policy)
        → executor (simulated rollback only)
        → PostgreSQL
      → read-only Kubernetes client
        → local cluster opspilot-dev
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
- Read-only discovery of Deployments, Pods, Services, and Events in `demo-shop`
- Idempotent sync of those workloads into OpsPilot services, labeled `source: kubernetes`
- `GET /health` and `GET /ready` (database and Kubernetes connectivity, no secrets)
- Development-only `POST /api/v1/demo/reset`, which restores `INC-142` and does not delete cluster data

### Simulated

- `production-01` incident telemetry, logs, traces, and the events embedded in `INC-142`
- The probable cause and its confidence
- The rollback executor and the health recovery that follows approval
- Reliability Lab runs

### Planned

- OpenTelemetry, Prometheus, and log/trace backends
- A real incident engine and a read-only investigator
- A policy engine with risk classes beyond the single allow rule
- A constrained Kubernetes remediation, still behind approval, for one named rollback
- Verification against live SLOs, then a learning loop
- GitOps and AWS/EKS

AWS, Terraform, Prometheus, Grafana, Argo, and an AI provider are intentionally not integrated. The Kubernetes client cannot create, update, delete, or exec.

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

Local cluster and demo workloads:

```bash
infra/scripts/up-dev-cluster.sh
```

The script creates `opspilot-dev` if needed and points kubeconfig at the node IP when the k3d load balancer cannot reach the API server.

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
docs         Architecture and ADRs
demo         Walkthrough for INC-142
infra        Optional container build
platform     Executor boundary notes
runbooks     payment-api rollback
```

## Roadmap

1. Ingest OpenTelemetry spans and Prometheus series for `demo-shop`, still read-only.
2. Replace the seeded analysis with a read-only investigator that can only emit a proposal.
3. Add a policy check with explicit deny reasons, still in front of a simulated executor.
4. Implement one Kubernetes action: roll back a single Deployment to a previous ReplicaSet, only after approval, then verify health.
5. Keep the investigator off the mutating client.

## Decisions

- [0001 — Monorepo](docs/adr/0001-monorepo.md)
- [0002 — Go control plane](docs/adr/0002-go-control-plane.md)
- [0003 — Safe remediation](docs/adr/0003-safe-remediation-model.md)
- [0004 — PostgreSQL persistence](docs/adr/0004-postgresql-persistence.md)
- [0005 — Read-only Kubernetes](docs/adr/0005-kubernetes-readonly.md)
- [0006 — Data provenance](docs/adr/0006-data-provenance.md)
