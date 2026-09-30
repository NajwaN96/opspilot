"""Sanitized, deterministic portfolio API.

Nothing here is live Kubernetes, Prometheus, Jaeger, PostgreSQL, or an executor.
POST is refused by the handler before this module is asked to mutate.
"""

from __future__ import annotations

READ_ONLY = (
    "This public portfolio is read-only. Remediation, experiments, and approvals "
    "stay on the local reliability lab."
)

STAMP = "2026-09-29T15:00:00Z"


def _service(service_id: str, name: str, status: str, availability: float, p95: int, error: float, version: str, previous: str) -> dict:
    return {
        "id": service_id,
        "name": name,
        "clusterId": "portfolio-sample",
        "namespace": "demo-shop",
        "status": status,
        "owner": "payments-platform",
        "runtime": "portfolio-sample",
        "description": f"Sanitized portfolio record for {name}. Not discovered from a cluster.",
        "version": version,
        "previousVersion": previous,
        "availability": availability,
        "p95LatencyMs": p95,
        "errorRate": error,
        "slo": {
            "objective": 99.9,
            "window": "portfolio-sample",
            "compliance": availability,
            "errorBudgetRemaining": 40 if status == "healthy" else 12,
            "burnRate": 0.2 if status == "healthy" else 4.1,
            "withinSLO": status == "healthy",
        },
        "lastDeployment": STAMP,
        "replicas": {"desired": 2, "ready": 2 if status != "critical" else 1},
        "dependencies": [],
        "deployments": [
            {
                "id": f"dep-{name}",
                "serviceId": service_id,
                "serviceName": name,
                "version": version,
                "previousVersion": previous,
                "at": STAMP,
                "clock": "15:00",
                "actor": "portfolio-sample",
                "status": "recorded",
                "strategy": "portfolio-sample",
                "change": "Sample release recorded for the public demo. Nothing was rolled out.",
            }
        ],
        "metrics": [
            {"clock": "14:40", "p95LatencyMs": 180, "errorRate": 0.4, "dbConnections": 30, "requestRate": 42},
            {"clock": "14:50", "p95LatencyMs": p95, "errorRate": error, "dbConnections": 55 if status != "healthy" else 32, "requestRate": 40},
            {"clock": "15:00", "p95LatencyMs": p95, "errorRate": error, "dbConnections": 70 if status != "healthy" else 31, "requestRate": 38},
        ],
        "openIncidents": ["INC-142"] if name == "payment-api" else [],
        "source": "portfolio-demo",
        "telemetry": "portfolio-sample",
        "image": "not-a-cluster-image",
        "workloadKind": "sample",
        "restarts": 0,
        "lastObserved": STAMP,
    }


def services() -> list[dict]:
    return [
        _service("k8s_demo-shop_payment-api", "payment-api", "degraded", 98.71, 1800, 12.4, "v1.8.2", "v1.8.1"),
        _service("k8s_demo-shop_checkout-api", "checkout-api", "healthy", 99.95, 140, 0.2, "v2.3.0", "v2.2.4"),
        _service("k8s_demo-shop_inventory", "inventory", "healthy", 99.99, 90, 0.05, "v1.4.2", "v1.4.1"),
    ]


def service(service_id: str) -> dict | None:
    for item in services():
        if item["id"] == service_id:
            return item
    return None


def incident() -> dict:
    return {
        "id": "INC-142",
        "severity": "sev-2",
        "serviceId": "k8s_demo-shop_payment-api",
        "serviceName": "payment-api",
        "title": "Portfolio sample: payment-api error budget burn after v1.8.2",
        "status": "investigating",
        "startedAt": STAMP,
        "durationSec": 900,
        "summary": "Sanitized sample of the detection, evidence, and approval story. Approving it here does nothing.",
        "cluster": "portfolio-sample",
        "origin": "portfolio-demo",
        "metricsSource": "portfolio-sample",
        "traceSource": "portfolio-sample",
        "kubernetesSource": "not-connected",
        "snapshot": {
            "availability": 98.71,
            "p95LatencyMs": 1800,
            "errorRate": 12.4,
            "dbConnections": 70,
            "status": "degraded",
            "version": "v1.8.2",
        },
        "timeline": [
            {"clock": "14:50", "at": STAMP, "title": "Sample deploy recorded", "detail": "v1.8.2 is a portfolio label, not a cluster rollout.", "kind": "deploy"},
            {"clock": "14:55", "at": STAMP, "title": "Sample symptom", "detail": "Error rate and latency in this record are fixed sample numbers.", "kind": "symptom"},
            {"clock": "15:00", "at": STAMP, "title": "Recommendation held", "detail": "Rollback stays a proposal. The public API refuses execution.", "kind": "recommendation"},
        ],
        "evidence": {
            "metrics": [
                {"clock": "14:50", "p95LatencyMs": 1800, "errorRate": 12.4, "dbConnections": 70, "requestRate": 40}
            ],
            "logs": [
                {"clock": "14:55", "level": "error", "service": "payment-api", "message": "sample: connection pool exhausted"}
            ],
            "traces": [
                {
                    "id": "trace-sample-1",
                    "traceId": "portfolio-trace-1",
                    "clock": "14:55",
                    "spans": [
                        {"id": "span-1", "service": "checkout-api", "name": "POST /checkout", "durationMs": 1900, "status": "error", "slow": True},
                        {"id": "span-2", "parentId": "span-1", "service": "payment-api", "name": "charge", "durationMs": 1700, "status": "error", "slow": True},
                    ],
                }
            ],
            "kubernetesEvents": [
                {"clock": "14:50", "type": "Normal", "reason": "Sample", "object": "portfolio/payment-api", "message": "Not read from a kube-apiserver."}
            ],
            "deployments": [],
        },
        "analysis": {
            "simulated": True,
            "cause": "Sample hypothesis: a connection leak shipped with payment-api v1.8.2.",
            "confidence": 91,
            "evidence": ["Fixed sample metric window", "Fixed sample trace"],
            "likelyCause": "Sample hypothesis only",
            "supporting": ["Error rate and pool usage move together in the sample"],
            "contradicting": ["No live cluster was queried"],
        },
        "recommendation": {
            "action": "rollback",
            "summary": "Sample proposal: roll back payment-api to v1.8.1. This copy cannot execute it.",
            "from": "v1.8.2",
            "to": "v1.8.1",
            "risk": "LOW",
            "policy": "Would require a person on the local lab",
            "allowed": True,
        },
    }


def telemetry(service_id: str) -> dict | None:
    item = service(service_id)
    if item is None:
        return None
    return {
        "source": "portfolio-demo",
        "available": True,
        "message": "Sanitized portfolio sample. Not a Prometheus query.",
        "service": item["name"],
        "namespace": "demo-shop",
        "updated": STAMP,
        "requestRate": 12.5,
        "errorRate": item["errorRate"] / 100,
        "requests": 240,
        "p50LatencyMs": 80 if item["status"] == "healthy" else 400,
        "p95LatencyMs": item["p95LatencyMs"],
        "p99LatencyMs": item["p95LatencyMs"] + 200,
        "availability": item["availability"] / 100,
        "dataSource": "portfolio-demo",
        "slo": {
            "target": 0.999,
            "window": "portfolio-sample",
            "label": "Sample window",
            "availability": item["availability"] / 100,
            "errorBudgetConsumed": 0.1 if item["status"] == "healthy" else 0.6,
            "burnRate": 0.2 if item["status"] == "healthy" else 4.1,
            "sufficient": True,
            "source": "portfolio-demo",
        },
    }


def traces(service_id: str) -> dict | None:
    if service(service_id) is None:
        return None
    return {
        "source": "portfolio-demo",
        "message": "Sanitized sample spans. Jaeger is not queried.",
        "traces": [
            {
                "id": "portfolio-trace-1",
                "start": STAMP,
                "durationMs": 1900,
                "rootService": "checkout-api",
                "status": "error",
                "spans": 2,
            }
        ],
    }


def trace_detail(trace_id: str) -> dict | None:
    if trace_id != "portfolio-trace-1":
        return None
    return {
        "id": trace_id,
        "start": STAMP,
        "durationMs": 1900,
        "rootService": "checkout-api",
        "status": "error",
        "spans": 2,
        "source": "portfolio-demo",
        "tree": [
            {"service": "checkout-api", "operation": "POST /checkout", "durationMs": 1900, "status": "error", "depth": 0},
            {"service": "payment-api", "operation": "charge", "durationMs": 1700, "status": "error", "depth": 1},
        ],
    }


def rollout() -> dict:
    return {
        "id": "ROL-PORTFOLIO-1",
        "service": "payment-api",
        "namespace": "demo-shop",
        "cluster": "portfolio-sample",
        "stableVersion": "1.4.2",
        "liveStableVersion": "1.4.2",
        "liveStableKnown": False,
        "liveStableReady": False,
        "candidateVersion": "1.5.0",
        "state": "AWAITING_APPROVAL",
        "weight": 5,
        "stageWeight": 5,
        "liveWeight": 0,
        "liveWeightKnown": False,
        "candidateActivity": "idle",
        "analysis": "PASS",
        "proposalAction": "promote",
        "aiAction": "continue",
        "aiSummary": "Sample recommendation only. It is not a gate and it was not produced by a live model call.",
        "aiStatus": "portfolio-sample",
        "aiMismatch": False,
        "verification": "Not executed. The public portfolio has no executor.",
        "requests": 40,
        "errorRate": 0.01,
        "p95": 0.18,
        "stableErrorRate": 0.004,
        "stableP95": 0.12,
        "candidateReady": True,
        "events": [
            {"at": STAMP, "title": "Sample canary recorded", "detail": "5 percent is a label in this record, not live traffic.", "kind": "info"}
        ],
    }


def cloud() -> dict:
    return {
        "accountState": "portfolio-demo",
        "accountMasked": "********3851",
        "region": "us-east-1",
        "authentication": "none-on-this-deployment",
        "cloudDeployment": "AWS — PORTFOLIO DEMO",
        "kubernetesRuntime": "Not connected. Local k3d lab is separate.",
        "paidCloudResources": "BLOCKED",
        "intentionalInfrastructureSpend": "$0",
        "eks": "plan-only",
    }


def kubernetes_status() -> dict:
    return {
        "cluster": "not-connected",
        "mode": "aws-portfolio-demo",
        "venue": "aws-portfolio-demo",
        "venueLabel": "AWS — PORTFOLIO DEMO",
        "namespace": "demo-shop",
        "namespaces": ["demo-shop"],
        "connectivity": "portfolio-demo",
        "kubernetesVersion": "not-connected",
        "lastSync": STAMP,
        "message": "Sanitized sample. This deployment is not connected to k3d or EKS.",
        "source": "portfolio-demo",
        "nodeCount": 0,
        "nodesReady": 0,
    }


def resolve(method: str, path: str):
    """Return (status, body). Mutations are always refused."""
    if method not in {"GET", "HEAD"}:
        return 403, {"error": {"code": "read_only", "message": READ_ONLY}}
    if path != "/" and path.endswith("/"):
        path = path[:-1]
    exact = {
        "/health": {"status": "ok", "service": "opspilot-portfolio", "version": "aws-portfolio-demo"},
        "/ready": {
            "status": "portfolio-demo",
            "database": "not-connected",
            "kubernetes": "not-connected",
            "demoResetEnabled": False,
            "prometheus": "not-connected",
            "opentelemetry": "not-connected",
            "traces": "not-connected",
        },
        "/api/v1/cloud/status": cloud(),
        "/api/v1/clusters": [
            {
                "id": "portfolio-sample",
                "name": "portfolio-sample",
                "environment": "aws-portfolio-demo",
                "region": "us-east-1",
                "kubernetesVersion": "not-connected",
                "status": "portfolio-demo",
                "provider": "sample",
                "simulated": True,
                "health": {
                    "state": "portfolio-demo",
                    "activeIncidents": 1,
                    "sloCompliance": 2,
                    "servicesWithinSLO": 2,
                    "servicesTotal": 3,
                    "servicesHealthy": 2,
                    "deploymentsToday": 0,
                },
                "nodes": [],
                "namespaces": [{"name": "demo-shop", "services": 3, "status": "portfolio-sample"}],
                "controlPlane": [],
            }
        ],
        "/api/v1/services": services(),
        "/api/v1/incidents": [incident()],
        "/api/v1/kubernetes/status": kubernetes_status(),
        "/api/v1/kubernetes/namespaces": [{"name": "demo-shop", "services": 3, "status": "portfolio-sample"}],
        "/api/v1/kubernetes/workloads": [
            {
                "name": "payment-api",
                "namespace": "demo-shop",
                "kind": "sample",
                "version": "v1.8.2",
                "image": "not-a-cluster-image",
                "desired": 2,
                "ready": 2,
                "restarts": 0,
                "status": "portfolio-sample",
                "lastObserved": STAMP,
                "serviceId": "k8s_demo-shop_payment-api",
                "source": "portfolio-demo",
            }
        ],
        "/api/v1/kubernetes/pods": [],
        "/api/v1/kubernetes/events": [
            {
                "id": "evt-portfolio-1",
                "at": STAMP,
                "namespace": "demo-shop",
                "type": "Normal",
                "reason": "Sample",
                "object": "portfolio/payment-api",
                "message": "Not read from a kube-apiserver.",
                "count": 1,
                "source": "portfolio-demo",
            }
        ],
        "/api/v1/telemetry/status": {
            "prometheus": "not-connected",
            "opentelemetry": "not-connected",
            "traces": "not-connected",
        },
        "/api/v1/rollouts": [rollout()],
        "/api/v1/experiments": {
            "simulated": True,
            "notice": "Experiments are not started from the public portfolio. The local lab owns fault injection.",
            "scenarios": [
                {
                    "id": "database-connection-exhaustion",
                    "name": "Database connection exhaustion",
                    "description": "Sample description only. This deployment cannot inject the fault.",
                    "target": "local lab",
                }
            ],
            "runs": [],
        },
    }
    if path in exact:
        return 200, exact[path]
    if path.startswith("/api/v1/services/") and path.endswith("/telemetry"):
        body = telemetry(path.removeprefix("/api/v1/services/").removesuffix("/telemetry"))
        return (200, body) if body else (404, {"error": {"message": "Unknown portfolio service"}})
    if path.startswith("/api/v1/services/") and path.endswith("/traces"):
        body = traces(path.removeprefix("/api/v1/services/").removesuffix("/traces"))
        return (200, body) if body else (404, {"error": {"message": "Unknown portfolio service"}})
    if path.startswith("/api/v1/services/"):
        body = service(path.removeprefix("/api/v1/services/"))
        return (200, body) if body else (404, {"error": {"message": "Unknown portfolio service"}})
    if path.startswith("/api/v1/incidents/") and path.endswith("/investigation"):
        incident_id = path.removeprefix("/api/v1/incidents/").removesuffix("/investigation")
        if incident_id != "INC-142":
            return 404, {"error": {"message": "Unknown portfolio incident"}}
        return 200, {
            "status": "portfolio-sample",
            "provider": "none",
            "model": "none",
            "real": False,
            "snapshotVersion": 1,
            "summary": "Sample investigation. No model was called and no key is present.",
            "likelyCauses": [
                {
                    "cause": "Sample hypothesis: connection leak in v1.8.2",
                    "confidence": 0.91,
                    "supporting_evidence_ids": ["ev-sample"],
                    "contradicting_evidence_ids": [],
                }
            ],
            "recommendedAction": {
                "action_type": "ROLLBACK_PAYMENT_API",
                "reason": "Sample label only. It does not select an image and it cannot execute.",
                "evidence_ids": ["ev-sample"],
            },
            "validation": {"accepted": True, "errors": []},
            "evidence": [
                {
                    "id": "ev-sample",
                    "source": "portfolio-demo",
                    "kind": "note",
                    "at": STAMP,
                    "title": "Sample evidence",
                    "body": "Fixed text. Not scraped from a cluster.",
                }
            ],
        }
    if path.startswith("/api/v1/incidents/"):
        if path.removeprefix("/api/v1/incidents/") == "INC-142":
            return 200, incident()
        return 404, {"error": {"message": "Unknown portfolio incident"}}
    if path.startswith("/api/v1/rollouts/"):
        if path.removeprefix("/api/v1/rollouts/") == "ROL-PORTFOLIO-1":
            return 200, rollout()
        return 404, {"error": {"message": "Unknown portfolio rollout"}}
    if path.startswith("/api/v1/traces/"):
        body = trace_detail(path.removeprefix("/api/v1/traces/"))
        return (200, body) if body else (404, {"error": {"message": "Unknown portfolio trace"}})
    return 404, {"error": {"message": "Not found"}}
