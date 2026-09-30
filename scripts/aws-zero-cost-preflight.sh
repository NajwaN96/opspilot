#!/usr/bin/env bash
# Fail closed before any zero-cost apply. Does not create resources.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
STACK="$ROOT/infra/aws-zero-cost"
PLAN="$STACK/tfplan"
# shellcheck source=lib/aws-session.sh
source "$ROOT/scripts/lib/aws-session.sh"
load_aws_session

if [[ "${OPSPILOT_ZERO_COST_DEPLOYMENT:-}" != "yes" ]]; then
  echo "BLOCKED BY ZERO-COST POLICY: set OPSPILOT_ZERO_COST_DEPLOYMENT=yes" >&2
  exit 1
fi

if [[ "${OPSPILOT_PAID_CLOUD_OVERRIDE:-}" == "yes" ]]; then
  echo "BLOCKED BY ZERO-COST POLICY: OPSPILOT_PAID_CLOUD_OVERRIDE must stay unset" >&2
  exit 1
fi

identity="$(aws --region us-east-1 --output json sts get-caller-identity)"
account="$(python3 -c 'import json,sys; print(json.loads(sys.argv[1])["Account"])' "$identity")"
if [[ ${#account} -ne 12 || "${account:8:4}" != "3851" ]]; then
  echo "BLOCKED BY ZERO-COST POLICY: unexpected account" >&2
  exit 1
fi

region="$(aws configure get region || true)"
if [[ "$region" != "us-east-1" ]]; then
  echo "BLOCKED BY ZERO-COST POLICY: region must be us-east-1" >&2
  exit 1
fi

plan_state="$(aws --region us-east-1 --output json freetier get-account-plan-state)"
python3 - "$plan_state" << 'PY'
import json, sys
plan = json.loads(sys.argv[1])
if plan.get("accountPlanType") != "FREE" or plan.get("accountPlanStatus") != "ACTIVE":
    raise SystemExit("BLOCKED BY ZERO-COST POLICY: account plan is not FREE and ACTIVE")
credits = (plan.get("accountPlanRemainingCredits") or {}).get("amount")
if credits is None:
    raise SystemExit("BLOCKED BY ZERO-COST POLICY: remaining credits were not readable")
print(f"free plan ACTIVE; remaining credits USD {credits}; expires {plan.get('accountPlanExpirationDate')}")
PY

if grep -R -n -E 'aws_eks_|aws_instance|aws_nat_gateway|aws_lb|aws_alb|aws_db_instance|aws_opensearch|aws_elasticache|aws_msk_|aws_eip|aws_s3_bucket|aws_cloudfront|aws_api_gateway|aws_apigateway|aws_dynamodb|aws_amplify|map_public_ip_on_launch' \
  --exclude-dir=.terraform --include='*.tf' "$STACK"; then
  echo "BLOCKED BY ZERO-COST POLICY: forbidden resource in the zero-cost stack" >&2
  exit 1
fi

"$STACK/package.sh"
cd "$STACK"
tofu init -input=false
tofu fmt -check
tofu validate
tofu plan -input=false -out="$PLAN" -var "package_path=$STACK/dist/portfolio.zip"
tofu show -json "$PLAN" > "$STACK/plan.json"
python3 "$STACK/guard.py" "$STACK/plan.json"
rm -f "$STACK/plan.json"
echo "preflight passed; plan saved at $PLAN"
