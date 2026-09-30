#!/usr/bin/env bash
# Build the static console and zip it with the read-only Lambda.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
HERE="$(cd "$(dirname "$0")" && pwd)"
WEB="$ROOT/apps/web"
SITE="$HERE/function/site"
DIST="$HERE/dist"
ZIP="$DIST/portfolio.zip"
SERVICE_PAGE="$WEB/src/app/services/[id]/page.tsx"
INCIDENT_PAGE="$WEB/src/app/incidents/[id]/page.tsx"
BACKUP="$(mktemp -d)"
cp "$SERVICE_PAGE" "$BACKUP/service.tsx"
cp "$INCIDENT_PAGE" "$BACKUP/incident.tsx"

restore() {
  cp "$BACKUP/service.tsx" "$SERVICE_PAGE"
  cp "$BACKUP/incident.tsx" "$INCIDENT_PAGE"
  rm -rf "$BACKUP"
}
trap restore EXIT

if ! grep -q 'export const dynamicParams = false;' "$SERVICE_PAGE"; then
  printf '\nexport const dynamicParams = false;\n' >> "$SERVICE_PAGE"
fi
if ! grep -q 'export const dynamicParams = false;' "$INCIDENT_PAGE"; then
  printf '\nexport const dynamicParams = false;\n' >> "$INCIDENT_PAGE"
fi

cd "$WEB"
OPSPILOT_AWS_PORTFOLIO=1 NEXT_PUBLIC_OPSPILOT_RUNTIME=aws-portfolio-demo npm run build

rm -rf "$SITE"
mkdir -p "$SITE" "$DIST"
cp -a "$WEB/out/." "$SITE/"
OPSPILOT_GIT_COMMIT="${OPSPILOT_GIT_COMMIT:-}" \
OPSPILOT_BUILD_TIME="${OPSPILOT_BUILD_TIME:-}" \
GITHUB_RUN_ID="${GITHUB_RUN_ID:-}" \
python3 - << PY
import hashlib
import os
import sys
import zipfile
from pathlib import Path

sys.path.insert(0, "$HERE")
import release_meta

here = Path("$HERE")
zip_path = Path("$ZIP")
if zip_path.exists():
    zip_path.unlink()
public = release_meta.public_labels(
    os.environ.get("OPSPILOT_GIT_COMMIT", ""),
    os.environ.get("OPSPILOT_BUILD_TIME", ""),
)
with zipfile.ZipFile(zip_path, "w", compression=zipfile.ZIP_DEFLATED) as archive:
    for name in ("handler.py", "catalog.py"):
        archive.write(here / "function" / name, name)
    archive.writestr("release.json", release_meta.dumps(public))
    site = here / "function" / "site"
    for path in site.rglob("*"):
        if path.is_file():
            archive.write(path, Path("site") / path.relative_to(site))
digest = hashlib.sha256(zip_path.read_bytes()).hexdigest()
(zip_path.parent / "portfolio.zip.sha256").write_text(f"{digest}  portfolio.zip\n", encoding="utf-8")
record = release_meta.artifact_record(public, digest, os.environ.get("GITHUB_RUN_ID", ""))
(zip_path.parent / "release-metadata.json").write_text(release_meta.dumps(record), encoding="utf-8")
print(zip_path.name, zip_path.stat().st_size, digest)
PY
