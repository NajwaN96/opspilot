# ADR 0027 — Zero-cost portfolio path

## Status

Accepted

## Context

The Phase 7 modules describe a real Amazon EKS development cluster. Official Amazon EKS pricing charges $0.10 per cluster-hour for a Kubernetes version in standard support, and $0.60 per cluster-hour in extended support. That page does not offer a free control plane. The same design launches one on-demand node (`t3.medium` by default; $0.0416 per hour for Linux in US East (N. Virginia) on the public EC2 price list), a 20 GiB node volume, and public IPv4 addresses at the published $0.005 per hour rate. Private ECR storage is free only inside a new-customer allowance, then billed, and this account's allowance cannot be read without credentials.

The portfolio requirement is $0 AWS cost. Credits are finite and were not visible from this environment. A resource with a non-zero price is not treated as free.

## Decision

Keep local k3d (`opspilot-dev`) as the only running Kubernetes environment. Keep the OpenTofu modules, GitOps overlays, and Argo CD application as reviewed Infrastructure-as-Code. Do not apply them.

`cloud-apply`, `ecr-build-push`, and `argocd-bootstrap` refuse when the Terraform tree contains EKS, a managed node group, Fargate, EC2, NAT, Elastic IP, a load balancer, RDS, ECR, or `map_public_ip_on_launch = true`. Refusal stands even if `OPSPILOT_CLOUD_APPROVED=yes`. A separate `OPSPILOT_PAID_CLOUD_OVERRIDE=yes` would bypass the cost check. Nothing in Terraform, CI, or those scripts sets it.

A later decision, [ADR 0028](0028-aws-zero-cost-portfolio.md), places the public console on an AWS Lambda function URL. Vercel is not used. The live SRE demonstration stays on local k3d, where GitOps desired state is already `gitops/overlays/local`. Argo CD is not installed on that cluster: the single node is the Phase 6 demo, and an in-cluster Argo install would compete with it. The Argo manifests remain validated and unused until a paid cloud is explicitly chosen.

IAM, the investigator, and the constrained executor are unchanged. The investigator still has no AWS credentials and no Kubernetes write client. `opspilot-aws-dev` is still not a mutation target.

## Consequences

`tofu fmt` and `tofu validate` remain useful. `tofu plan` needs credentials and does not create infrastructure, but it is not required for the zero-cost path. `cloud-destroy` stays available so a future paid stack can be removed. No AWS resource is created by adopting this decision.
