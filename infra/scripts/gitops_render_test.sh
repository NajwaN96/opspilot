#!/usr/bin/env bash
# Render both overlays and check the cloud image rewrite and namespace set.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
LOCAL="$(kustomize build "$ROOT/gitops/overlays/local")"
AWS="$(kustomize build "$ROOT/gitops/overlays/aws-dev")"

printf '%s\n' "$LOCAL" | grep -q "kind: Deployment" || {
  echo "local overlay did not render a Deployment" >&2
  exit 1
}
printf '%s\n' "$LOCAL" | grep -q "image: opspilot-demo:" || {
  echo "local overlay lost the local image name" >&2
  exit 1
}
if printf '%s\n' "$LOCAL" | grep -q "ecr.invalid"; then
  echo "local overlay points at the cloud placeholder registry" >&2
  exit 1
fi

printf '%s\n' "$AWS" | grep -q "image: ecr.invalid/opspilot-demo:" || {
  echo "aws overlay did not rewrite opspilot-demo" >&2
  exit 1
}

printf '%s\n' "$AWS" >/tmp/opspilot-aws-render.yaml
python3 - << 'PY'
from pathlib import Path
text = Path("/tmp/opspilot-aws-render.yaml").read_text()
namespaces = set()
current = None
for line in text.splitlines():
    if line.startswith("kind:"):
        current = line.split(":", 1)[1].strip()
    if line.strip().startswith("namespace:") and current not in (None, "Namespace"):
        namespaces.add(line.split(":", 1)[1].strip())
allowed = {"demo-shop", "opspilot-system"}
extra = namespaces - allowed
if extra:
    raise SystemExit(f"unexpected namespaces: {sorted(extra)}")
if "demo-shop" not in namespaces or "opspilot-system" not in namespaces:
    raise SystemExit(f"missing expected namespaces: {sorted(namespaces)}")
PY

grep -q "repoURL: https://github.com/example/opspilot" "$ROOT/gitops/argocd/applications/opspilot-demo.yaml"
grep -q "path: gitops/overlays/aws-dev" "$ROOT/gitops/argocd/applications/opspilot-demo.yaml"
grep -q "selfHeal: true" "$ROOT/gitops/argocd/applications/opspilot-demo.yaml"
grep -q "prune: false" "$ROOT/gitops/argocd/applications/opspilot-demo.yaml"
if grep -q "admin.password\|argocd-initial-admin-secret" "$ROOT/gitops/argocd/applications/opspilot-demo.yaml"; then
  echo "Argo application must not carry admin credentials" >&2
  exit 1
fi
echo "gitops render passed"
