#!/usr/bin/env bash
# Replace the code of the existing portfolio Lambda. Does not apply infrastructure.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
ZIP="$ROOT/infra/aws-zero-cost/dist/portfolio.zip"

if [[ "${1:-}" == "--dry-run" ]]; then
  python3 "$ROOT/infra/aws-zero-cost/delivery.py" publish --zip "$ZIP" --dry-run
  exit 0
fi

if [[ "${OPSPILOT_PORTFOLIO_PUBLISH:-}" != "yes" ]]; then
  echo "BLOCKED: set OPSPILOT_PORTFOLIO_PUBLISH=yes to update the portfolio Lambda" >&2
  exit 1
fi

if [[ "${OPSPILOT_PAID_CLOUD_OVERRIDE:-}" == "yes" ]]; then
  echo "BLOCKED BY ZERO-COST POLICY: paid override must stay unset" >&2
  exit 1
fi

if [[ "${OPSPILOT_GITHUB_OIDC:-}" != "yes" ]]; then
  # shellcheck source=lib/aws-session.sh
  source "$ROOT/scripts/lib/aws-session.sh"
  load_aws_session
fi

exec python3 "$ROOT/infra/aws-zero-cost/delivery.py" publish --zip "$ZIP"
