#!/usr/bin/env bash
# Static checks that do not need AWS. Fails if secrets or cloud mutation paths appear.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

if grep -R -n -E 'AKIA[0-9A-Z]{16}|ASIA[0-9A-Z]{16}|BEGIN [A-Z ]*PRIVATE KEY' \
  --exclude=security_scan.sh \
  --exclude-dir=.git --exclude-dir=node_modules --exclude-dir=.next --exclude-dir=.terraform \
  infra gitops apps docs .github >/tmp/secret-scan.txt; then
  echo "possible credential in the tree:" >&2
  cat /tmp/secret-scan.txt >&2
  exit 1
fi

if grep -R -n -E 'aws-sdk-go|github.com/aws/aws-sdk|argoproj|hashicorp/aws' \
  apps/api/internal/investigate >/tmp/ai-cloud.txt; then
  echo "investigator package references a cloud client:" >&2
  cat /tmp/ai-cloud.txt >&2
  exit 1
fi

if grep -R -n -E 'HandleFunc\("POST /api/v1/(kubectl|apply|patch|shell)' apps/api >/tmp/mutation-api.txt; then
  echo "generic mutation route is registered:" >&2
  cat /tmp/mutation-api.txt >&2
  exit 1
fi

echo "security scan passed"
