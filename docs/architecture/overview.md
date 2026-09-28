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
| `internal/config` | Address, database URL, environment, namespaces, sync interval |

## Target path

```
Kubernetes
  → OpenTelemetry
  → Prometheus / logs / traces
  → OpsPilot intake
  → incident engine
  → investigator
  → remediation proposal
  → policy engine
  → human approval when required
  → constrained Kubernetes action
  → recovery verification
```

Intake is read-only. It normalizes deploy markers, metrics, logs, traces, and Kubernetes events into the same evidence model the incident page already renders.

The incident engine opens and updates incidents from those signals. It does not change the cluster.

The investigator reads an incident and its evidence and returns a proposal: action, target, from version, to version, confidence, and the evidence it used. It does not receive a Kubernetes client.

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
- No mutating Kubernetes client and no kubectl endpoint
- No Loki, Tempo, or Grafana
- No model API key
- No Terraform or Argo CD
- No browser-supplied PromQL, shell command, or experiment namespace

Those are later adapters. They attach at intake, policy, or the executor. They should not be threaded through the React views.
