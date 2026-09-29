# ADR 0024 — GitOps with Kustomize and Argo CD

## Status

Accepted

## Context

The local cluster is still applied by `infra/scripts/up-dev-cluster.sh`. The cloud cluster needs a desired state in Git and a reconciler that is not a standing kubectl session.

## Decision

`gitops/base` holds the demo-shop and observability manifests. `gitops/overlays/local` is what k3d applies. `gitops/overlays/aws-dev` rewrites `opspilot-demo` to `ecr.invalid/opspilot-demo` until the bootstrap substitutes the real ECR hostname. Third-party images stay on their public registries because the nodes have outbound internet.

Argo CD v3.5.3 is installed by `argocd-bootstrap` into the `argocd` namespace. It is not installed on k3d. The Application `opspilot-demo` syncs `gitops/overlays/aws-dev` with automated self-heal and with prune left off, so a replica change is restored and a hand-created pod is not deleted. The admin password is the in-cluster initial secret and is not committed. Access is a port-forward, not a public load balancer.

`gitops-drift-demo` pauses auto-sync, scales `payment-api` by one replica, shows OutOfSync, syncs back, and turns self-heal on again.

## Consequences

The cloud image name is not real until ECR exists and the overlay is updated. Argo needs a Git URL it can read, passed as `OPSPILOT_GITOPS_REPO`. A private origin needs a read-only deploy key created outside this repository.
