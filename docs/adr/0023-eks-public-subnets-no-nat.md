# ADR 0023 — Public subnets and no NAT gateway

## Status

Accepted

## Context

A NAT gateway has an hourly charge and a data-processing charge that dominate a one-node portfolio cluster. Private-only nodes cannot pull images or reach the EKS API without either NAT or a set of interface VPC endpoints, and those endpoints are also billed per hour.

## Decision

The VPC is `10.42.0.0/16` with two public subnets, one internet gateway, and a default route to that gateway. There is no `aws_nat_gateway`. Node subnets set `map_public_ip_on_launch`, so the managed nodes get public IPv4 addresses for egress. The node group does not open SSH. Inbound node access stays on the security group EKS attaches, which is not `0.0.0.0/0`.

The EKS API is public and private. `api_cidrs` must list specific administrator networks. `0.0.0.0/0` fails validation.

The tradeoff is that worker nodes have public addresses. Egress is simple and the NAT charge is avoided. The control plane is not open to the internet beyond the configured CIDRs.

## Consequences

A public IPv4 address on the node is a small hourly charge. Anyone who can reach a node port that the security group allows can talk to it, so the group must stay closed to the world. Adding a private subnet later is a deliberate change, not a default.
