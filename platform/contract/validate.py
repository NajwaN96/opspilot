"""Validate the OpsPilot service contract.

Catalog files are JSON documents. JSON is valid YAML 1.2, which is the
on-disk form of service.yaml. The validator rejects malformed definitions.
"""

from __future__ import annotations

import json
from pathlib import Path

CHECKS = (
    "sloDefined",
    "runbook",
    "owner",
    "healthEndpoint",
    "readinessProbe",
    "metrics",
    "tracing",
    "resourceLimits",
    "ci",
    "gitops",
    "securityContext",
)
VERDICTS = {"PASS", "WARNING", "MISSING"}
STRATEGIES = {"rolling", "canary"}
REQUIRED_SPEC = (
    "description",
    "owner",
    "team",
    "runtime",
    "language",
    "repository",
    "environment",
    "namespace",
    "version",
    "slo",
    "dependencies",
    "runbook",
    "dashboard",
    "onCall",
    "deploymentStrategy",
    "checks",
)


def repo_root() -> Path:
    return Path(__file__).resolve().parents[2]


def load_json(path: Path) -> dict:
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
    except json.JSONDecodeError as exc:
        raise ValueError(f"{path.name} is not valid JSON") from exc
    if not isinstance(data, dict):
        raise ValueError(f"{path.name} must be an object")
    return data


def validate_service(service: dict, runbook_ids: set[str]) -> list[str]:
    reasons = []
    if service.get("apiVersion") != "opspilot.io/v1" or service.get("kind") != "Service":
        reasons.append("service apiVersion/kind must be opspilot.io/v1 Service")
    name = (service.get("metadata") or {}).get("name")
    if not isinstance(name, str) or not name:
        reasons.append("metadata.name is required")
    spec = service.get("spec")
    if not isinstance(spec, dict):
        return reasons + ["spec is required"]
    for field in REQUIRED_SPEC:
        if field not in spec:
            reasons.append(f"spec.{field} is required")
    strategy = spec.get("deploymentStrategy")
    if strategy not in STRATEGIES:
        reasons.append("deploymentStrategy must be rolling or canary")
    slo = spec.get("slo") or {}
    if not isinstance(slo, dict) or not isinstance(slo.get("availability"), (int, float)):
        reasons.append("spec.slo.availability is required")
    if not isinstance(slo.get("latencyMs"), (int, float)):
        reasons.append("spec.slo.latencyMs is required")
    deps = spec.get("dependencies")
    if not isinstance(deps, list) or any(not isinstance(item, str) for item in deps):
        reasons.append("dependencies must be a list of names")
    runbook = spec.get("runbook")
    if isinstance(runbook, str) and runbook not in runbook_ids:
        reasons.append(f"runbook {runbook} is not in the library")
    checks = spec.get("checks")
    if not isinstance(checks, dict):
        reasons.append("checks are required")
    else:
        for key in CHECKS:
            if checks.get(key) not in VERDICTS:
                reasons.append(f"checks.{key} must be PASS, WARNING, or MISSING")
    return reasons


def validate_tree(root: Path | None = None) -> list[str]:
    root = root or repo_root()
    catalog = load_json(root / "platform" / "catalog" / "catalog.json")
    library = load_json(root / "platform" / "runbooks" / "runbooks.json")
    reasons = []
    if catalog.get("kind") != "ServiceCatalog":
        reasons.append("catalog kind must be ServiceCatalog")
    services = catalog.get("services")
    runbooks = library.get("runbooks")
    if not isinstance(services, list) or not services:
        return reasons + ["catalog has no services"]
    if not isinstance(runbooks, list) or not runbooks:
        return reasons + ["runbook library is empty"]
    ids = []
    for book in runbooks:
        if not isinstance(book, dict) or not book.get("id"):
            reasons.append("runbook missing id")
            continue
        ids.append(book["id"])
        for field in ("title", "service", "symptoms", "detection", "evidence", "likelyCauses", "safeInvestigation", "allowedRemediation", "verification", "escalation"):
            if field not in book:
                reasons.append(f"runbook {book['id']} missing {field}")
    seen = set()
    for service in services:
        name = (service.get("metadata") or {}).get("name")
        if name in seen:
            reasons.append(f"duplicate service {name}")
        seen.add(name)
        reasons.extend(validate_service(service, set(ids)))
    expected = {"storefront", "checkout-api", "payment-api", "orders-api", "inventory-api"}
    if seen != expected:
        reasons.append("catalog must list the demo-shop services exactly once")
    return reasons
