#!/usr/bin/env bash
# Create the local opspilot-dev cluster and the demo-shop workloads.
# Safe to re-run. Does not touch cloud accounts.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
CLUSTER="${OPSPILOT_K8S_CLUSTER:-opspilot-dev}"

if ! k3d cluster list | awk 'NR>1 {print $1}' | grep -qx "$CLUSTER"; then
  k3d cluster create "$CLUSTER" \
    --servers 1 \
    --agents 0 \
    --image rancher/k3s:v1.31.4-k3s1 \
    --wait \
    --timeout 300s \
    --k3s-arg "--disable=traefik@server:0" \
    --k3s-arg "--disable-network-policy@server:0" \
    --k3s-arg "--snapshotter=native@server:0" \
    --k3s-arg "--flannel-backend=host-gw@server:0"
fi

# The k3d load balancer sometimes cannot reach the API server on nested Docker.
# Talk to the node IP directly when it answers and the published proxy does not.
NODE_IP="$(docker inspect "k3d-${CLUSTER}-server-0" --format '{{.NetworkSettings.Networks.k3d-'"${CLUSTER}"'.IPAddress}}')"
if [[ -n "$NODE_IP" ]]; then
  kubectl config set-cluster "k3d-${CLUSTER}" --server="https://${NODE_IP}:6443" >/dev/null
fi

kubectl --request-timeout=30s wait --for=condition=Ready node --all --timeout=180s
kubectl apply -f "$ROOT/infra/kubernetes/demo-shop.yaml"
kubectl --request-timeout=30s -n demo-shop rollout status deployment/storefront deployment/checkout-api deployment/payment-api deployment/orders-api deployment/inventory-api --timeout=180s
kubectl -n demo-shop get deploy,po
