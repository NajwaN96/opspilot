# Architecture

OpsPilot is a reliability control plane. It sits beside a cluster, not inside it as a privileged controller. The MVP runs entirely on one machine with simulated operational data so the incident workflow can be reviewed before any credential exists.

## Today

```
┌────────────┐    HTTP     ┌──────────────────────────────┐
│ Next.js UI │ ──────────► │ Go API                       │
│ apps/web   │  /api/v1    │  handlers                    │
└────────────┘             │  service (policy)            │
                           │    ├─ executor (simulated)   │
                           │    └─ repository (memory)    │
                           └──────────────────────────────┘
```

The browser talks only to the Next.js server. That server proxies `/api/*` and `/health` to the Go process. The Go process is the system of record for the session.

Package boundaries in `apps/api`:

| Package | Responsibility |
| --- | --- |
| `internal/httpapi` | Routes, JSON, request logs, CORS |
| `internal/service` | Use cases. Authorizes a remediation before the executor runs |
| `internal/executor` | Performs one constrained action. The simulated implementation records it and returns |
| `internal/repository` | `Catalog` interface |
| `internal/repository/memory` | Seeded `production-01` world and the in-memory remediation clock |
| `internal/model` | Shared response types |
| `internal/config` | Address, CORS origins, remediation step interval |

Handlers do not know whether the data is fake. Replacing the memory store with PostgreSQL, or the simulated executor with a Kubernetes client, is a constructor change in `cmd/opspilot`.

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

- No cloud provider SDK
- No kubeconfig and no in-cluster client
- No Prometheus, Loki, Tempo, or collector
- No model API key
- No Terraform
- No persisted audit log yet (the approval exists only in process memory)

Those are later adapters. They attach at intake, policy, or the executor. They should not be threaded through the React views.
