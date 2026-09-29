#!/usr/bin/env bash
# Guardrail unit checks. No AWS calls.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
# shellcheck source=lib/guard.sh
source "$ROOT/infra/scripts/lib/guard.sh"

expect_fail() {
  local name="$1"
  shift
  if "$@" >/tmp/guard-out 2>/tmp/guard-err; then
    echo "expected failure: $name" >&2
    exit 1
  fi
}

expect_ok() {
  local name="$1"
  shift
  if ! "$@" >/tmp/guard-out 2>/tmp/guard-err; then
    echo "expected success: $name" >&2
    cat /tmp/guard-err >&2
    exit 1
  fi
}

expect_ok dev guard_require_dev_environment dev
expect_fail production guard_require_dev_environment production
expect_fail staging guard_require_dev_environment staging
expect_fail empty-env guard_require_dev_environment ""

expect_ok account guard_require_account 123456789012 123456789012
expect_fail mismatch guard_require_account 123456789012 210987654321
expect_fail placeholder guard_require_account 000000000000 000000000000
expect_fail short guard_require_account 123 123456789012

expect_ok region guard_require_region eu-west-1 eu-west-1 us-east-1
expect_fail other-region guard_require_region ap-southeast-2 eu-west-1
expect_fail gov guard_require_region us-gov-west-1 us-gov-west-1

expect_ok cluster guard_require_cluster opspilot-aws-dev
expect_fail local-cluster guard_require_cluster opspilot-dev
expect_fail prod-cluster guard_require_cluster opspilot-prod

OPSPILOT_CLOUD_APPROVED=yes expect_ok approval guard_require_approval
OPSPILOT_CLOUD_APPROVED= expect_fail no-approval guard_require_approval
OPSPILOT_CLOUD_APPROVED=yes OPSPILOT_CLOUD_DESTROY=yes expect_ok destroy guard_require_destroy_approval
OPSPILOT_CLOUD_APPROVED=yes OPSPILOT_CLOUD_DESTROY= expect_fail destroy-without-flag guard_require_destroy_approval

if grep -R "aws_nat_gateway" "$ROOT/infra/terraform" >/dev/null; then
  echo "NAT gateway is not allowed in the portfolio stack" >&2
  exit 1
fi
if grep -R "AdministratorAccess" "$ROOT/infra/terraform" >/dev/null; then
  echo "AdministratorAccess is not allowed" >&2
  exit 1
fi

echo "guard tests passed"
