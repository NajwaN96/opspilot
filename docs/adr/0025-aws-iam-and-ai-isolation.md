# ADR 0025 — IAM boundary and AI isolation

## Status

Accepted

## Context

EKS needs IAM roles for the control plane and the nodes. OpsPilot workloads do not call AWS APIs. The investigator must not gain a cloud client.

## Decision

The cluster role can be assumed only by `eks.amazonaws.com` and is attached to `AmazonEKSClusterPolicy`. The node role can be assumed only by `ec2.amazonaws.com` and is attached to the worker-node, VPC-CNI, and ECR read-only policies. Those three let the node join the cluster and pull images. There is no administrator policy, no IRSA role, and no EKS Pod Identity association. None of the demo pods receive an AWS credential.

The cluster creator gets EKS access-entry admin so the operator who applied the stack can use kubectl. That principal is the human running `cloud-apply`, not the API process and not the investigator.

`apps/api/internal/investigate` does not import an AWS SDK, Argo CD, or Terraform. Rollout approval still requires `cluster == opspilot-dev`. Pointing the same API at `opspilot-aws-dev` does not enable the payment-api rollback or the canary executor. There is no generic apply, patch, kubectl, or shell route.

## Consequences

Image pull uses the node role. If a future workload needs an AWS API, it gets its own role. It does not reuse the node role, and the investigator still does not receive it.
