# ADR 0026 — Cost footprint and teardown

## Status

Accepted

## Decision

The default footprint is one EKS control plane on Kubernetes 1.35, one `t3.medium` node, a 20 GiB node volume, two public subnets, an internet gateway, and one ECR repository with scan-on-push and a lifecycle rule that keeps the last 10 tagged images. There is no NAT gateway, no RDS, no ElastiCache, no MSK, no OpenSearch, no managed Prometheus or Grafana, and no load balancer. Control-plane logs are left off so CloudWatch ingestion is not added by default. ECR tags are mutable because the demo workflow republishes fixed tags such as `1.4.2`. A lifecycle policy caps accumulation.

Node size, min, desired, and max are variables. The maximum is 3, and the instance type must be `t3.small`, `t3.medium`, or `t3.large`. `t3.small` is allowed but is tight for the observability limits.

`cloud-destroy` requires `OPSPILOT_CLOUD_APPROVED=yes` and `OPSPILOT_CLOUD_DESTROY=yes`. It deletes only load balancers tagged `Project=OpsPilot` and `Environment=dev`, then runs `tofu destroy` for this state. It does not delete unrelated account resources. ECR repositories are created with `force_delete` so images do not block the repository.

## Execution policy

This footprint is portfolio Infrastructure-as-Code. It is not the default deployment. Amazon EKS has a published hourly control-plane price and is not an always-free service. The managed node, its EBS volume, and `map_public_ip_on_launch` can also bill. `infra/scripts/cost-classify` fails closed on that surface, and `cloud-apply`, `ecr-build-push`, and `argocd-bootstrap` call the same check before any AWS or registry call. `OPSPILOT_PAID_CLOUD_OVERRIDE` is a separate explicit switch and is not enabled.

## Consequences

If the paid override were ever set, the standing cost would be the EKS control plane, the node, the node volume, and the public IPv4 address. Destroy remains part of the design. `cloud-destroy` is still guarded by the account, region, and destroy flags so an accidental stack can be removed. It is not run automatically.
