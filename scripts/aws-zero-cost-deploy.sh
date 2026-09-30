#!/usr/bin/env bash
# Apply the already-planned zero-cost stack. Preflight is mandatory.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
if [[ "${OPSPILOT_ZERO_COST_DEPLOYMENT:-}" != "yes" ]]; then
  echo "BLOCKED BY ZERO-COST POLICY: set OPSPILOT_ZERO_COST_DEPLOYMENT=yes" >&2
  exit 1
fi
"$ROOT/scripts/aws-zero-cost-preflight.sh"
# shellcheck source=lib/aws-session.sh
source "$ROOT/scripts/lib/aws-session.sh"
load_aws_session
cd "$ROOT/infra/aws-zero-cost"
tofu apply -input=false tfplan
tofu output -raw portfolio_url
echo
