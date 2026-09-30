"""Lambda Function URL entrypoint for the public portfolio.

Serves the static Next.js export and the read-only catalog. It has no AWS
credentials of its own beyond the execution role used for its own log stream,
and that role is not exposed to the browser.
"""

from __future__ import annotations

import base64
import json
import mimetypes
from pathlib import Path

import catalog

ROOT = Path(__file__).resolve().parent / "site"
MUTATION = {
    "error": {
        "code": "read_only",
        "message": catalog.READ_ONLY,
    }
}

EXTRA_TYPES = {
    ".js": "text/javascript; charset=utf-8",
    ".mjs": "text/javascript; charset=utf-8",
    ".css": "text/css; charset=utf-8",
    ".json": "application/json; charset=utf-8",
    ".html": "text/html; charset=utf-8",
    ".txt": "text/plain; charset=utf-8",
    ".svg": "image/svg+xml",
    ".woff2": "font/woff2",
    ".map": "application/json",
    ".webp": "image/webp",
    ".ico": "image/x-icon",
}


def handler(event, _context):
    method = ((event or {}).get("requestContext") or {}).get("http", {}).get("method") or "GET"
    method = method.upper()
    raw_path = (event or {}).get("rawPath") or "/"
    if method not in {"GET", "HEAD"}:
        return _json(403, MUTATION)
    if raw_path.startswith("/api/") or raw_path in {"/health", "/ready"}:
        status, body = catalog.resolve(method, raw_path)
        return _json(status, {} if method == "HEAD" else body)
    return _static(method, raw_path)


def _json(status: int, body: dict) -> dict:
    encoded = json.dumps(body).encode("utf-8")
    return {
        "statusCode": status,
        "headers": {
            "content-type": "application/json; charset=utf-8",
            "cache-control": "no-store",
            "x-content-type-options": "nosniff",
            "x-opspilot-runtime": "aws-portfolio-demo",
        },
        "body": encoded.decode("utf-8"),
        "isBase64Encoded": False,
    }


def _static(method: str, raw_path: str) -> dict:
    relative = raw_path.split("?", 1)[0]
    if relative.startswith("/"):
        relative = relative[1:]
    candidate = _safe(relative)
    if candidate is None:
        return _html_status(400, "Bad path")
    file_path = _locate(candidate)
    if file_path is None:
        return _html_status(404, "Not found")
    data = file_path.read_bytes()
    if len(data) > 5_500_000:
        return _html_status(500, "Asset exceeds the Lambda response limit")
    content_type = EXTRA_TYPES.get(file_path.suffix.lower()) or mimetypes.guess_type(file_path.name)[0] or "application/octet-stream"
    cache = "public, max-age=31536000, immutable" if "/_next/static/" in raw_path else "no-cache"
    return {
        "statusCode": 200,
        "headers": {
            "content-type": content_type,
            "cache-control": cache,
            "x-content-type-options": "nosniff",
            "x-opspilot-runtime": "aws-portfolio-demo",
        },
        "body": "" if method == "HEAD" else base64.b64encode(data).decode("ascii"),
        "isBase64Encoded": method != "HEAD",
    }


def _safe(relative: str) -> Path | None:
    if relative == "":
        return Path(".")
    path = Path(relative)
    if path.is_absolute() or ".." in path.parts:
        return None
    return path


def _locate(relative: Path) -> Path | None:
    root = ROOT.resolve()
    direct = (root / relative).resolve()
    if not _inside(root, direct):
        return None
    if direct.is_file():
        return direct
    for suffix in ("index.html", "index.txt"):
        nested = (direct / suffix).resolve()
        if _inside(root, nested) and nested.is_file():
            return nested
    html = direct.with_suffix(direct.suffix + ".html") if direct.suffix else Path(str(direct) + ".html")
    html = html.resolve()
    if _inside(root, html) and html.is_file():
        return html
    return None


def _inside(root: Path, candidate: Path) -> bool:
    try:
        candidate.relative_to(root)
    except ValueError:
        return False
    return True


def _html_status(status: int, message: str) -> dict:
    body = f"<!doctype html><title>{message}</title><p>{message}</p>".encode("utf-8")
    return {
        "statusCode": status,
        "headers": {"content-type": "text/html; charset=utf-8", "cache-control": "no-store"},
        "body": base64.b64encode(body).decode("ascii"),
        "isBase64Encoded": True,
    }
