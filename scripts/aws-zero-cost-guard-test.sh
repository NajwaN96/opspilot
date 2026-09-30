#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
python3 -m unittest discover -s "$ROOT/infra/aws-zero-cost" -p 'test_guard.py'
python3 -m unittest discover -s "$ROOT/infra/aws-zero-cost/function" -p 'test_*.py'
echo "zero-cost guard tests passed"
