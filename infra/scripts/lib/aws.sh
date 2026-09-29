#!/usr/bin/env bash
# Read-only AWS identity helpers. Never print access keys or session tokens.

aws_caller_account() {
  aws sts get-caller-identity --query Account --output text
}

aws_caller_arn() {
  aws sts get-caller-identity --query Arn --output text
}

aws_configured_region() {
  if [[ -n "${AWS_REGION:-}" ]]; then
    printf '%s\n' "$AWS_REGION"
    return 0
  fi
  if [[ -n "${AWS_DEFAULT_REGION:-}" ]]; then
    printf '%s\n' "$AWS_DEFAULT_REGION"
    return 0
  fi
  local configured
  configured="$(aws configure get region 2>/dev/null || true)"
  if [[ -n "$configured" ]]; then
    printf '%s\n' "$configured"
    return 0
  fi
  return 1
}
