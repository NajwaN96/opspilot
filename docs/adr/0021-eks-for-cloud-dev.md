# ADR 0021 — EKS for the cloud dev cluster

## Status

Accepted

## Context

Phases 1–6 run on a local k3d cluster named `opspilot-dev`. Phase 7 needs a real cloud Kubernetes control plane without replacing that local environment.

## Decision

The cloud dev cluster is Amazon EKS, named `opspilot-aws-dev`. The Kubernetes version defaults to 1.35, which is in EKS standard support. Version 1.31 matches the local k3s node but is in extended support and is rejected by the stack, because extended support is billed at a higher control-plane rate.

The local cluster stays on k3s v1.31.4. The API labels `opspilot-dev` as `LOCAL — LIVE` and `opspilot-aws-dev` as `AWS — PLAN ONLY`. Other cluster names are `UNKNOWN`. The constrained executor still allows only `opspilot-dev`. EKS is not deployed.

## Consequences

The portfolio pays for one EKS control plane and a small managed node group. Local development does not need AWS credentials.
