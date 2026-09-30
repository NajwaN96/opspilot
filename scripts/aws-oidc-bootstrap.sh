#!/usr/bin/env bash
# Create the GitHub OIDC provider and the portfolio deploy role.
# Refuses to run until OPSPILOT_GITHUB_REPOSITORY is one verified owner/name.
# Does not print the account id and does not apply the EKS stack.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
# shellcheck source=lib/aws-session.sh
source "$ROOT/scripts/lib/aws-session.sh"

if [[ "${OPSPILOT_ZERO_COST_DEPLOYMENT:-}" != "yes" ]]; then
  echo "BLOCKED BY ZERO-COST POLICY: set OPSPILOT_ZERO_COST_DEPLOYMENT=yes" >&2
  exit 1
fi

if [[ "${OPSPILOT_PAID_CLOUD_OVERRIDE:-}" == "yes" ]]; then
  echo "BLOCKED BY ZERO-COST POLICY: paid override must stay unset" >&2
  exit 1
fi

python3 - "${OPSPILOT_GITHUB_REPOSITORY:-}" << 'PY'
import re, sys
repo = sys.argv[1]
if not re.fullmatch(r"[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+", repo) or "*" in repo or ".." in repo.split("/"):
    raise SystemExit("BLOCKED: OPSPILOT_GITHUB_REPOSITORY must be one verified owner/name")
PY

# GitHub includes owner and repository ids in the OIDC subject. A name-only
# subject cannot assume the role. Do not opt the repository out of that claim.
subject_json="$(gh api "repos/${OPSPILOT_GITHUB_REPOSITORY}/actions/oidc/customization/sub")"
subject_prefix="$(python3 - "$subject_json" << 'PY'
import json, re, sys
doc = json.loads(sys.argv[1])
if not doc.get("use_immutable_subject"):
    raise SystemExit("BLOCKED: GitHub immutable OIDC subjects must stay enabled")
prefix = doc.get("sub_claim_prefix") or ""
if not re.fullmatch(r"repo:[A-Za-z0-9_.-]+@[0-9]+/[A-Za-z0-9_.-]+@[0-9]+", prefix) or "*" in prefix:
    raise SystemExit("BLOCKED: GitHub subject prefix is not a single repository")
print(prefix)
PY
)"

load_aws_session
identity="$(aws --region us-east-1 --output json sts get-caller-identity)"
account="$(python3 -c 'import json,sys; print(json.loads(sys.argv[1])["Account"])' "$identity")"
if [[ ${#account} -ne 12 || "${account:8:4}" != "3851" ]]; then
  echo "BLOCKED BY ZERO-COST POLICY: unexpected account" >&2
  exit 1
fi

plan_state="$(aws --region us-east-1 --output json freetier get-account-plan-state)"
python3 - "$plan_state" << 'PY'
import json, sys
plan = json.loads(sys.argv[1])
if plan.get("accountPlanType") != "FREE" or plan.get("accountPlanStatus") != "ACTIVE":
    raise SystemExit("BLOCKED BY ZERO-COST POLICY: account plan is not FREE and ACTIVE")
print("free plan ACTIVE")
PY

provider_arn="arn:aws:iam::${account}:oidc-provider/token.actions.githubusercontent.com"
if aws iam get-open-id-connect-provider --open-id-connect-provider-arn "$provider_arn" >/dev/null 2>&1; then
  echo "GitHub OIDC provider already present"
else
  aws iam create-open-id-connect-provider \
    --url https://token.actions.githubusercontent.com \
    --client-id-list sts.amazonaws.com \
    --thumbprint-list 6938fd4d98bab03faadb97b34396831e3780aea1 1c58a3a8518e8759bf075b76b750d4f2df264fcd \
    >/dev/null
  echo "GitHub OIDC provider created"
fi

trust="$(mktemp)"
trap 'rm -f "$trust"' EXIT
ACCOUNT="$account" REPO="${OPSPILOT_GITHUB_REPOSITORY}" SUBJECT_PREFIX="$subject_prefix" TRUST_FILE="$trust" python3 - << PY
import json, os, sys
sys.path.insert(0, "$ROOT/infra/aws-zero-cost")
import ci_policy
policy = json.loads(open("$ROOT/infra/aws-zero-cost/iam/github-deployer-policy.json", encoding="utf-8").read())
reasons = ci_policy.scan_iam_policy(policy)
if reasons:
    raise SystemExit("BLOCKED: " + "; ".join(reasons))
rendered = ci_policy.render_trust(
    os.environ["ACCOUNT"],
    os.environ["REPO"],
    subject_prefix=os.environ["SUBJECT_PREFIX"],
)
open(os.environ["TRUST_FILE"], "w", encoding="utf-8").write(json.dumps(rendered))
PY

if aws iam get-role --role-name opspilot-github-deployer >/dev/null 2>&1; then
  aws iam update-assume-role-policy \
    --role-name opspilot-github-deployer \
    --policy-document "file://${trust}" >/dev/null
  echo "updated trust on opspilot-github-deployer"
else
  aws iam create-role \
    --role-name opspilot-github-deployer \
    --assume-role-policy-document "file://${trust}" \
    --description "Deploy the existing OpsPilot portfolio Lambda from GitHub Actions" \
    >/dev/null
  echo "created role opspilot-github-deployer"
fi

aws iam put-role-policy \
  --role-name opspilot-github-deployer \
  --policy-name opspilot-portfolio-deploy \
  --policy-document "file://${ROOT}/infra/aws-zero-cost/iam/github-deployer-policy.json" \
  >/dev/null
echo "inline policy opspilot-portfolio-deploy applied"
echo "Set GitHub variable OPSPILOT_AWS_ROLE_ARN to the ARN of role opspilot-github-deployer."
echo "The account id is not printed."
