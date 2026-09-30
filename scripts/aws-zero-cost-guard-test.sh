#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
(
  cd "$ROOT/infra/aws-zero-cost"
  python3 -m unittest test_guard test_delivery test_ci_policy test_release_meta
)
(
  cd "$ROOT/infra/aws-zero-cost/function"
  python3 -m unittest test_catalog
)
echo "zero-cost guard tests passed"
