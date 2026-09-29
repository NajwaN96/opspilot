#!/usr/bin/env bash
# Load dev tfvars and refuse the wrong account, region, or environment.
# Does not create AWS resources.

cloud_root() {
  cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd
}

cloud_tfvars() {
  printf '%s\n' "$(cloud_root)/infra/terraform/environments/dev/dev.auto.tfvars"
}

cloud_stack_dir() {
  printf '%s\n' "$(cloud_root)/infra/terraform/environments/dev"
}

tfvar_string() {
  local key="$1"
  local file="$2"
  sed -n "s/^${key}[[:space:]]*=[[:space:]]*\"\\([^\"]*\\)\".*/\\1/p" "$file" | head -1
}

cloud_preflight() {
  local mode="${1:-plan}"
  local root file caller region allowed_region tf_region tf_account tf_env
  root="$(cloud_root)"
  # shellcheck source=guard.sh
  source "$root/infra/scripts/lib/guard.sh"
  # shellcheck source=aws.sh
  source "$root/infra/scripts/lib/aws.sh"

  file="$(cloud_tfvars)"
  [[ -f "$file" ]] || {
    echo "cloud guard: missing $file" >&2
    return 1
  }
  tf_region="$(tfvar_string aws_region "$file")"
  tf_account="$(tfvar_string allowed_account_id "$file")"
  tf_env="$(tfvar_string environment "$file")"
  [[ -n "$tf_region" && "$tf_region" != "REPLACE_ME" ]] || {
    echo "cloud guard: aws_region is not set in tfvars" >&2
    return 1
  }
  guard_require_dev_environment "$tf_env" || return 1
  guard_require_cluster "opspilot-aws-dev" || return 1
  guard_require_region "$tf_region" "$tf_region" || return 1

  caller="$(aws_caller_account)"
  guard_require_account "$caller" "$tf_account" || return 1
  region="$(aws_configured_region)" || {
    echo "cloud guard: no AWS region is configured" >&2
    return 1
  }
  [[ "$region" == "$tf_region" ]] || {
    echo "cloud guard: configured region ${region} does not match tfvars ${tf_region}" >&2
    return 1
  }

  if [[ "$mode" == "apply" ]]; then
    guard_require_approval || return 1
  fi
  if [[ "$mode" == "destroy" ]]; then
    guard_require_destroy_approval || return 1
  fi

  echo "account: $caller"
  echo "arn: $(aws_caller_arn)"
  echo "region: $region"
  echo "environment: $tf_env"
  echo "cluster: opspilot-aws-dev"
  echo "mode: $mode"
}
