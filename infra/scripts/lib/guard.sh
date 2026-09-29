#!/usr/bin/env bash
# Account, region, and environment checks shared by the cloud scripts.
# This file is sourced. It does not talk to AWS by itself.

guard_refuse() {
  echo "cloud guard: $*" >&2
  return 1
}

guard_require_dev_environment() {
  local env="${1:-}"
  case "$env" in
    dev) ;;
    *)
      guard_refuse "environment must be dev (got ${env:-empty})"
      return 1
      ;;
  esac
  case "$env" in
    *prod* | *production* | *stage* | *staging*)
      guard_refuse "environment name is not a dev environment"
      return 1
      ;;
  esac
}

guard_require_account() {
  local caller="${1:-}"
  local allowed="${2:-}"
  [[ "$caller" =~ ^[0-9]{12}$ ]] || {
    guard_refuse "caller account is not a 12-digit id"
    return 1
  }
  [[ "$allowed" =~ ^[0-9]{12}$ ]] || {
    guard_refuse "allowed account is not configured"
    return 1
  }
  [[ "$allowed" != "000000000000" ]] || {
    guard_refuse "allowed account is still the example placeholder"
    return 1
  }
  [[ "$caller" == "$allowed" ]] || {
    guard_refuse "caller account does not match the allowlist"
    return 1
  }
}

guard_require_region() {
  local region="${1:-}"
  shift || true
  local allowed ok=0
  [[ -n "$region" ]] || {
    guard_refuse "region is empty"
    return 1
  }
  case "$region" in
    *gov* | cn-* | *china*)
      guard_refuse "region is outside the commercial portfolio set"
      return 1
      ;;
  esac
  for allowed in "$@"; do
    if [[ "$region" == "$allowed" ]]; then
      ok=1
    fi
  done
  [[ "$ok" -eq 1 ]] || {
    guard_refuse "region ${region} is not in the allowlist"
    return 1
  }
}

guard_require_cluster() {
  local name="${1:-}"
  [[ "$name" == "opspilot-aws-dev" ]] || {
    guard_refuse "cluster must be opspilot-aws-dev"
    return 1
  }
}

guard_require_approval() {
  [[ "${OPSPILOT_CLOUD_APPROVED:-}" == "yes" ]] || {
    guard_refuse "OPSPILOT_CLOUD_APPROVED=yes is required. Approval is never inferred."
    return 1
  }
}

guard_require_destroy_approval() {
  guard_require_approval || return 1
  [[ "${OPSPILOT_CLOUD_DESTROY:-}" == "yes" ]] || {
    guard_refuse "OPSPILOT_CLOUD_DESTROY=yes is also required before destroy."
    return 1
  }
}
