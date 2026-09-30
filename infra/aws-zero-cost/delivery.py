"""Update the existing portfolio Lambda and restore the previous zip on failure.

No infrastructure apply. No published Lambda versions. No customer S3 bucket.
The previous package is the zip Lambda already stored for the function.
"""

from __future__ import annotations

import json
import sys
import urllib.error
import urllib.request
import zipfile
from io import BytesIO

FUNCTION_NAME = "opspilot-portfolio"
REGION = "us-east-1"
ACCOUNT_SUFFIX = "3851"

GET_ROUTES = (
    "/",
    "/services",
    "/incidents",
    "/infrastructure",
    "/rollouts",
    "/settings",
    "/health",
    "/api/v1/cloud/status",
    "/api/v1/release",
)
POST_ROUTES = (
    "/api/v1/demo/reset",
    "/api/v1/incidents/INC-142/remediations",
)
RELEASE_KEYS = frozenset({"version", "gitCommit", "buildTime", "environment"})


class DeliveryError(Exception):
    pass


def check_context(*, account_id: str, region: str, function_name: str, plan: dict, override: str | None) -> list[str]:
    reasons = []
    if override == "yes":
        reasons.append("paid override is set")
    if not (isinstance(account_id, str) and len(account_id) == 12 and account_id.isdigit() and account_id.endswith(ACCOUNT_SUFFIX)):
        reasons.append("wrong account")
    if region != REGION:
        reasons.append("wrong region")
    if function_name != FUNCTION_NAME:
        reasons.append("wrong Lambda name")
    plan_type = (plan or {}).get("accountPlanType")
    plan_status = (plan or {}).get("accountPlanStatus")
    if plan_type != "FREE" or plan_status != "ACTIVE":
        reasons.append("account plan is not FREE and ACTIVE")
    return reasons


def validate_artifact(payload: bytes) -> None:
    if not isinstance(payload, (bytes, bytearray)) or len(payload) < 200:
        raise DeliveryError("bad artifact")
    if len(payload) > 45_000_000:
        raise DeliveryError("artifact exceeds the direct Lambda upload limit")
    try:
        with zipfile.ZipFile(BytesIO(payload)) as archive:
            names = set(archive.namelist())
    except zipfile.BadZipFile as exc:
        raise DeliveryError("bad artifact") from exc
    missing = {"handler.py", "catalog.py", "release.json"} - names
    if missing:
        raise DeliveryError("artifact is missing " + ", ".join(sorted(missing)))


def evaluate_smoke(samples: list[tuple[str, str, int, str]]) -> list[str]:
    """samples are (method, path, status, body)."""
    reasons = []
    seen = {(method, path) for method, path, _, _ in samples}
    for path in GET_ROUTES:
        if ("GET", path) not in seen:
            reasons.append(f"missing GET {path}")
    for path in POST_ROUTES:
        if ("POST", path) not in seen:
            reasons.append(f"missing POST {path}")
    for method, path, status, body in samples:
        if method == "GET" and path in GET_ROUTES and status != 200:
            reasons.append(f"GET {path} returned {status}")
        if method == "POST" and path in POST_ROUTES and status != 403:
            reasons.append(f"POST {path} returned {status}")
        if path == "/api/v1/cloud/status" and "PORTFOLIO DEMO" not in (body or ""):
            reasons.append("cloud status is not AWS — PORTFOLIO DEMO")
        if path == "/api/v1/release" and method == "GET" and status == 200:
            reasons.extend(_release_reasons(body))
    return reasons


def _release_reasons(body: str) -> list[str]:
    try:
        payload = json.loads(body or "")
    except json.JSONDecodeError:
        return ["release body is not JSON"]
    if not isinstance(payload, dict):
        return ["release body is not an object"]
    extra = set(payload) - RELEASE_KEYS
    reasons = []
    if extra:
        reasons.append("release returned extra fields")
    if payload.get("environment") != "aws-portfolio-demo":
        reasons.append("release environment is not aws-portfolio-demo")
    return reasons


def smoke(client) -> list[str]:
    samples = []
    for path in GET_ROUTES:
        status, body = client.request("GET", path)
        samples.append(("GET", path, status, body))
    for path in POST_ROUTES:
        status, body = client.request("POST", path)
        samples.append(("POST", path, status, body))
    return evaluate_smoke(samples)


def publish_artifact(client, payload: bytes) -> None:
    reasons = check_context(**client.context())
    if reasons:
        raise DeliveryError("refusing to deploy: " + "; ".join(reasons))
    validate_artifact(payload)
    previous = client.previous_package()
    validate_artifact(previous)
    client.update_package(payload)
    client.wait_until_updated()
    try:
        failed = smoke(client)
    except Exception as exc:
        failed = [f"smoke request failed ({type(exc).__name__})"]
    if not failed:
        return
    client.update_package(previous)
    client.wait_until_updated()
    try:
        rolled = smoke(client)
    except Exception as exc:
        rolled = [f"rollback smoke failed ({type(exc).__name__})"]
    if rolled:
        raise DeliveryError("verification failed and rollback verification failed")
    raise DeliveryError("verification failed; previous package restored")


def _aws(*args: str) -> dict:
    import subprocess

    completed = subprocess.run(
        ["aws", "--region", REGION, "--output", "json", *args],
        check=True,
        capture_output=True,
        text=True,
    )
    if not completed.stdout.strip():
        return {}
    return json.loads(completed.stdout)


class AwsLambda:
    """CLI adapter. Presigned code URLs are never logged."""

    def context(self) -> dict:
        identity = _aws("sts", "get-caller-identity")
        plan = _aws("freetier", "get-account-plan-state")
        import os

        return {
            "account_id": identity.get("Account", ""),
            "region": os.environ.get("AWS_REGION") or os.environ.get("AWS_DEFAULT_REGION") or "",
            "function_name": FUNCTION_NAME,
            "plan": plan,
            "override": os.environ.get("OPSPILOT_PAID_CLOUD_OVERRIDE"),
        }

    def previous_package(self) -> bytes:
        described = _aws("lambda", "get-function", "--function-name", FUNCTION_NAME)
        location = ((described.get("Code") or {}).get("Location")) or ""
        if not location.startswith("https://"):
            raise DeliveryError("previous package URL was not returned")
        with urllib.request.urlopen(location, timeout=60) as response:  # noqa: S310 - Lambda-issued URL
            return response.read()

    def update_package(self, payload: bytes) -> None:
        import subprocess
        import tempfile

        with tempfile.NamedTemporaryFile(suffix=".zip") as handle:
            handle.write(payload)
            handle.flush()
            subprocess.run(
                [
                    "aws",
                    "--region",
                    REGION,
                    "lambda",
                    "update-function-code",
                    "--function-name",
                    FUNCTION_NAME,
                    "--zip-file",
                    "fileb://" + handle.name,
                ],
                check=True,
                capture_output=True,
                text=True,
            )

    def wait_until_updated(self) -> None:
        import subprocess

        subprocess.run(
            ["aws", "--region", REGION, "lambda", "wait", "function-updated", "--function-name", FUNCTION_NAME],
            check=True,
            capture_output=True,
            text=True,
        )

    def request(self, method: str, path: str) -> tuple[int, str]:
        url = _function_url() + path
        req = urllib.request.Request(url, method=method)
        try:
            with urllib.request.urlopen(req, timeout=20) as response:
                return response.status, response.read().decode("utf-8", "replace")
        except urllib.error.HTTPError as exc:
            return exc.code, exc.read().decode("utf-8", "replace")


def _function_url() -> str:
    described = _aws("lambda", "get-function-url-config", "--function-name", FUNCTION_NAME)
    url = described.get("FunctionUrl") or ""
    if not url.startswith("https://") or not url.endswith(".lambda-url.us-east-1.on.aws/"):
        raise DeliveryError("portfolio function URL is not the expected host")
    return url[:-1]


def _read_zip(path: str) -> bytes:
    from pathlib import Path

    return Path(path).read_bytes()


def main(argv: list[str]) -> int:
    if len(argv) < 2 or argv[1] not in {"check", "publish", "smoke"}:
        print("usage: delivery.py check | publish --zip FILE [--dry-run] | smoke --url URL", file=sys.stderr)
        return 2
    if argv[1] == "check":
        reasons = check_context(**AwsLambda().context())
        if reasons:
            print("BLOCKED: " + "; ".join(reasons), file=sys.stderr)
            return 1
        print("portfolio deploy context accepted")
        return 0
    if argv[1] == "smoke":
        if len(argv) != 4 or argv[2] != "--url":
            print("usage: delivery.py smoke --url URL", file=sys.stderr)
            return 2
        base = argv[3].rstrip("/")
        client = _UrlSmoke(base)
        reasons = smoke(client)
        if reasons:
            print("SMOKE FAILED", file=sys.stderr)
            for reason in reasons:
                print(reason, file=sys.stderr)
            return 1
        print("smoke passed")
        return 0
    if "--zip" not in argv:
        print("usage: delivery.py publish --zip FILE [--dry-run]", file=sys.stderr)
        return 2
    payload = _read_zip(argv[argv.index("--zip") + 1])
    validate_artifact(payload)
    if "--dry-run" in argv:
        print("dry-run accepted; Lambda was not updated")
        return 0
    try:
        publish_artifact(AwsLambda(), payload)
    except DeliveryError as exc:
        print(str(exc), file=sys.stderr)
        return 1
    print("portfolio Lambda updated and verified")
    return 0


class _UrlSmoke:
    def __init__(self, base: str) -> None:
        self.base = base

    def request(self, method: str, path: str) -> tuple[int, str]:
        req = urllib.request.Request(self.base + path, method=method)
        try:
            with urllib.request.urlopen(req, timeout=20) as response:
                return response.status, response.read().decode("utf-8", "replace")
        except urllib.error.HTTPError as exc:
            return exc.code, exc.read().decode("utf-8", "replace")


if __name__ == "__main__":
    raise SystemExit(main(sys.argv))
