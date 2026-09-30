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
python3 - << PY
import zipfile
from pathlib import Path
here = Path("$HERE")
zip_path = Path("$ZIP")
if zip_path.exists():
    zip_path.unlink()
with zipfile.ZipFile(zip_path, "w", compression=zipfile.ZIP_DEFLATED) as archive:
    for name in ("handler.py", "catalog.py"):
        archive.write(here / "function" / name, name)
    site = here / "function" / "site"
    for path in site.rglob("*"):
        if path.is_file():
            archive.write(path, Path("site") / path.relative_to(site))
print(zip_path, zip_path.stat().st_size)
PY
