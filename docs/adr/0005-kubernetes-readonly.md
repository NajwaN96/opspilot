# ADR 0005 — Read-only Kubernetes discovery

## Status

Accepted

## Context

Phase 2 needs a real local cluster so the console can show actual Deployments and Pods. The remediation path is not ready to mutate that cluster. An investigator must not receive a client that can apply manifests or run commands.

## Decision

The Go API depends on a `kubernetes.Reader` interface. The implementation uses client-go and exposes only list and get calls: cluster version, nodes, namespaces, Deployments, Pods, and Events.

Discovery is limited to `OPSPILOT_K8S_NAMESPACES` (default `demo-shop`). `opspilot-system` may appear in the namespace list. Other namespaces are not ingested, and a request for a workload outside the allow-list is rejected.

There is no endpoint that accepts a shell command, a kubeconfig body, or a manifest to apply. The simulated executor still does not call Kubernetes.

The local cluster is k3d, named `opspilot-dev`. It is not EKS and it is not created by Terraform.

## Consequences

- Workload health in the UI can be real while incident metrics stay simulated. Provenance fields keep those apart.
- A later mutating executor must be a different type, constructed only after policy and approval, and it must not be passed to the investigator. ADR 0011 adds that type for one payment-api rollback. It is not part of `Reader`.
- If kubeconfig is missing, the API stays up and reports the cluster as disconnected. It does not invent workloads.
