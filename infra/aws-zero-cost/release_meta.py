"""Sanitized build labels for the portfolio artifact.

The public record never includes an account id, a caller ARN, a filesystem
path, or a credential. Extra keys supplied by the caller are dropped.
"""

from __future__ import annotations

import json
import re

COMMIT_RE = re.compile(r"[0-9a-f]{7,40}")
TIME_RE = re.compile(r"[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9:]+Z")
VERSION_RE = re.compile(r"[A-Za-z0-9._-]{1,40}")
RUN_RE = re.compile(r"[0-9]{1,12}")


def public_labels(commit: str, built: str, version: str = "aws-portfolio-demo") -> dict[str, str]:
    if not COMMIT_RE.fullmatch(commit or ""):
        commit = "unknown"
    if not TIME_RE.fullmatch(built or ""):
        built = "unknown"
    if not VERSION_RE.fullmatch(version or ""):
        version = "aws-portfolio-demo"
    return {
        "version": version,
        "gitCommit": commit,
        "buildTime": built,
        "environment": "aws-portfolio-demo",
    }


def artifact_record(public: dict[str, str], digest: str, workflow_run: str) -> dict[str, str]:
    if not re.fullmatch(r"[0-9a-f]{64}", digest or ""):
        raise ValueError("artifact checksum must be a sha256 hex digest")
    run = workflow_run if RUN_RE.fullmatch(workflow_run or "") else "local"
    labels = public_labels(public.get("gitCommit", ""), public.get("buildTime", ""), public.get("version", ""))
    return {
        "version": labels["version"],
        "gitCommit": labels["gitCommit"],
        "buildTime": labels["buildTime"],
        "artifactSha256": digest,
        "workflowRun": run,
        "environment": "aws-portfolio-demo",
        "functionName": "opspilot-portfolio",
        "region": "us-east-1",
    }


def dumps(payload: dict[str, str]) -> str:
    return json.dumps(payload, indent=2, sort_keys=True) + "\n"
