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

import_image() {
  local image="$1"
  if ! docker image inspect "$image" >/dev/null 2>&1; then
    docker pull "$image"
  fi
  docker save "$image" | docker exec -i "k3d-${CLUSTER}-server-0" ctr -n k8s.io images import -
}

# CoreDNS is required for service DNS. In-cluster pulls time out here.
import_image rancher/mirrored-coredns-coredns:1.12.0
kubectl -n kube-system delete pod -l k8s-app=kube-dns --wait=false >/dev/null 2>&1 || true
kubectl --request-timeout=30s -n kube-system rollout status deployment/coredns --timeout=180s

import_image opspilot-demo:0.3.0
import_image prom/prometheus:v2.55.1
import_image otel/opentelemetry-collector:0.115.1
import_image jaegertracing/all-in-one:1.62.0

kubectl apply -f "$ROOT/infra/kubernetes/observability.yaml"
kubectl apply -f "$ROOT/infra/kubernetes/demo-shop.yaml"
kubectl --request-timeout=30s -n opspilot-system rollout status deployment/prometheus deployment/otel-collector deployment/jaeger --timeout=180s
kubectl --request-timeout=30s -n demo-shop rollout status deployment/storefront deployment/checkout-api deployment/payment-api deployment/orders-api deployment/inventory-api deployment/traffic --timeout=180s
kubectl -n demo-shop get deploy,po
kubectl -n opspilot-system get deploy,po
