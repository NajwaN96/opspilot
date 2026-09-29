#!/usr/bin/env bash
# Classify Terraform that can create a non-zero AWS charge.
# Sourced by cloud scripts. Does not call AWS and does not apply.

cost_scan_dir() {
  local dir="$1"
  [[ -d "$dir" ]] || return 0
  grep -R -n -E \
    -e 'resource[[:space:]]+"aws_(eks_cluster|eks_node_group|eks_fargate_profile|instance|nat_gateway|eip|lb|alb|elb|db_instance|ecr_repository)"' \
    -e 'map_public_ip_on_launch[[:space:]]*=[[:space:]]*true' \
    --include='*.tf' --exclude-dir=.terraform "$dir" || true
}

guard_require_zero_cost() {
  local root="$1"
  local hits
  hits="$(cost_scan_dir "$root/infra/terraform")"
  if [[ -z "$hits" ]]; then
    return 0
  fi
  if [[ "${OPSPILOT_PAID_CLOUD_OVERRIDE:-}" == "yes" ]]; then
    echo "cloud guard: OPSPILOT_PAID_CLOUD_OVERRIDE=yes is set. Billable resources would be allowed." >&2
    echo "cloud guard: this portfolio does not enable that override." >&2
    return 0
  fi
  echo "cloud guard: refusing deployment. These definitions can charge the AWS account:" >&2
  echo "$hits" >&2
  echo "cloud guard: zero-cost mode is the default. EKS, nodes, public IPv4, and ECR stay plan-only." >&2
  echo "cloud guard: do not set OPSPILOT_PAID_CLOUD_OVERRIDE." >&2
  return 1
}
