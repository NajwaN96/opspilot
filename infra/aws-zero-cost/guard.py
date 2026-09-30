"""Fail-closed classifier for the zero-cost OpenTofu plan.

OPSPILOT_ZERO_COST_DEPLOYMENT does not make a forbidden resource legal.
The caller must already have refused to run unless that flag is yes.
"""

from __future__ import annotations

import json
import sys
from pathlib import Path

ALLOWED_RESOURCES = {
    "aws_cloudwatch_log_group",
    "aws_iam_role",
    "aws_iam_role_policy",
    "aws_lambda_function",
    "aws_lambda_function_url",
    "aws_lambda_permission",
}

ALLOWED_DATA = {
    "aws_caller_identity",
    "aws_region",
}

FORBIDDEN_MARKERS = (
    "aws_eks_",
    "aws_instance",
    "aws_nat_gateway",
    "aws_lb",
    "aws_alb",
    "aws_db_instance",
    "aws_opensearch",
    "aws_elasticache",
    "aws_msk_",
    "aws_eip",
    "aws_s3_bucket",
    "aws_cloudfront_distribution",
    "aws_api_gateway",
    "aws_apigateway",
    "aws_dynamodb",
    "aws_amplify",
    "aws_ecs_",
    "aws_ecr_",
    "map_public_ip_on_launch",
)


def resource_types(plan: dict) -> list[str]:
    found = []
    for change in plan.get("resource_changes") or []:
        actions = ((change.get("change") or {}).get("actions")) or []
        if actions == ["no-op"]:
            continue
        found.append(change.get("type") or "")
    return found


def data_types(plan: dict) -> list[str]:
    found = []
    for item in (plan.get("configuration") or {}).get("root_module", {}).get("resources") or []:
        mode = item.get("mode")
        if mode == "data":
            found.append(item.get("type") or "")
    return found


def reject_reasons(plan: dict) -> list[str]:
    reasons = []
    blob = json.dumps(plan)
    for marker in FORBIDDEN_MARKERS:
        if marker in blob:
            reasons.append(f"forbidden marker {marker}")
    for kind in resource_types(plan):
        if kind not in ALLOWED_RESOURCES:
            reasons.append(f"resource {kind or 'unknown'} is not on the zero-cost allowlist")
    for kind in data_types(plan):
        if kind not in ALLOWED_DATA:
            reasons.append(f"data source {kind or 'unknown'} is not on the zero-cost allowlist")
    return reasons


def main(argv: list[str]) -> int:
    if len(argv) != 2:
        print("usage: guard.py PLAN.json", file=sys.stderr)
        return 2
    plan = json.loads(Path(argv[1]).read_text())
    reasons = reject_reasons(plan)
    if reasons:
        print("BLOCKED BY ZERO-COST POLICY", file=sys.stderr)
        for reason in reasons:
            print(reason, file=sys.stderr)
        return 1
    print("zero-cost plan accepted")
    for kind in sorted(set(resource_types(plan))):
        print(kind)
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv))
