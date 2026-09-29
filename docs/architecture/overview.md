# Architecture

OpsPilot is a reliability control plane. It sits beside a cluster, not inside it as a privileged controller. `INC-142` analysis and rollback stay simulated. demo-shop metrics, traces, detection, and the payment-api fault are real and local.

## Today

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
                                                │
demo-shop                                       │
    │                                           │
    ├── Metrics ────────────────────────────────┘
    │
    └── OTLP
         │
         ▼
 OpenTelemetry Collector
         │
         ▼
    Jaeger
         │
         └──────────────→ OpsPilot

Reliability Lab
      │
      ▼
Constrained Experiment Controller
      │
      ▼
payment-api degraded mode
      │
      ▼
Real telemetry degradation
      │
      ▼
Detection Engine
      │
      ▼
Real Incident
      │
      ▼
Evidence Correlation
      │
      ▼
Rule-Based Diagnosis
```

The browser talks only to the Next.js server. That server proxies `/api/*`, `/health`, and `/ready` to the Go process. PostgreSQL is the system of record for approvals and discovered workloads. The simulated clock still renders `INC-142` telemetry, rehydrated from `demo_state` after a restart.

Package boundaries in `apps/api`:

| Package | Responsibility |
| --- | --- |
| `internal/httpapi` | Routes, JSON, request IDs, CORS |
| `internal/service` | Use cases. Authorizes a remediation before the executor runs |
| `internal/executor` | Performs one constrained action. The simulated implementation records it and returns |
| `internal/repository` | `Catalog` interface |
| `internal/repository/memory` | Seeded `production-01` world and the remediation clock |
| `internal/repository/postgres` | Migrations, durable approvals, audit, Kubernetes observations |
| `internal/kubernetes` | Read-only client, mapping, sync, and a tiny allow-listed service proxy. No mutate methods |
| `internal/telemetry` | Prometheus and Jaeger readers. PromQL is built in this package |
| `internal/detection` | Thresholds, deduplication streaks, diagnosis, and the local SLO |
| `internal/experiment` | Allow-listed payment-api fault. It does not roll back a Deployment |
| `internal/model` | Shared response types, including source and telemetry |
| `internal/release` | The only payment-api version pair the executor may apply |
| `internal/investigate` | Evidence snapshots, sanitization, provider calls, and citation checks. No executor and no Kubernetes client |
| `internal/config` | Address, database URL, environment, namespaces, sync interval, AI provider |

## Investigation path

```
Real incident
  → Evidence builder
       Prometheus
       OpenTelemetry / Jaeger
       Kubernetes reader
       Change metadata
       Runbooks
  → Evidence snapshot
  → Sanitization
  → OpenAI investigator
  → Structured output
  → Evidence validation
  → Semantic recommendation
  → Deterministic proposal builder
  → Policy engine
  → Human approval
  → Constrained executor
  → Kubernetes
  → Prometheus verification
```

The investigator reads the snapshot. It does not receive the mutator, a shell, kubectl, a Docker socket, arbitrary SQL, or arbitrary PromQL. `ROLLBACK_PAYMENT_API` is a semantic label. `release.Propose` still decides the image pair, and only after a person approves does the executor change `demo-shop/payment-api`.

## Security principle

An AI component must never receive unrestricted Kubernetes administrative access.

Future control flow:

```
AI recommendation
  → deterministic policy validation
  → risk classification
  → human approval when required
  → constrained executor
  → verification
```

Policy is code, not a prompt. It checks the action name, the object, the version pair, the risk class, and whether a person has approved. The executor is a small interface. A Kubernetes implementation may, for example, patch one Deployment's pod template back to a known ReplicaSet. It must not wrap a cluster-admin REST config, and it must not accept a free-form manifest from the investigator.

The MVP already follows that split for one case. `POST /api/v1/incidents/INC-142/remediations` with `{"action":"rollback"}` is allowed because the seeded recommendation says so. Any other action is rejected before the executor is called. The executor logs the request and does not shell out.

## What is deliberately absent

- No cloud provider SDK and no EKS
- No generic kubectl, apply, patch, or exec endpoint
- No Loki or Grafana
- No API key in the browser, Compose, or Kubernetes manifests
- No Terraform or Argo CD
- No browser-supplied PromQL, shell command, or experiment namespace
- No autonomous remediation and no approval bypass

The payment-api mutator is a separate method from the reader. The investigator is not given it.
