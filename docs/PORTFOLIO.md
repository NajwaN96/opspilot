# OpsPilot in an interview

## Problem

A bad release shows up as a chart, a trace, and a chat message. The person who can roll it back often has broad cluster access, and the reason for the rollback is reconstructed afterwards.

## Solution

OpsPilot is a reliability control plane. It detects a payment-api regression from Prometheus, attaches traces and the Deployment image, asks for a human approval, and runs one allowlisted rollback. Everything else stays read-only.

## Architecture

Three venues, kept separate:

- **Local k3d `opspilot-dev`.** Go API, PostgreSQL, demo-shop, Prometheus, OpenTelemetry, Jaeger, Grafana, Alertmanager. This is the live lab.
- **Public Lambda.** The same console shape, sanitized data, GitHub OIDC deploy, no access keys. POST returns 403.
- **EKS in OpenTofu.** Plan only. CI validates it. It has never been applied.

## Engineering decisions

- The executor knows one action: `demo-shop/payment-api` from `opspilot-demo:1.5.0-bad` back to `opspilot-demo:1.4.2`.
- The AI investigator receives a sanitized evidence snapshot. It cannot call Kubernetes, a shell, or PromQL.
- Canaries move 5%, 25%, 50%, 100% only when the Prometheus gate allows it. A bad candidate aborts onto the stable image.
- GitHub deploys the portfolio with OIDC and a role that can update that one Lambda and read the free-plan state.
- The service catalog is a validated contract. The scorecard reports missing security contexts instead of inventing a score.

## Reliability

Detection is deterministic. Verification reads Prometheus again after the rollback. The audit trail records approval, execution, and the result.

## Security

Public mutation is blocked. Discovery is read-only. Remediation is allowlisted. Secrets, kubeconfig, and Terraform state are not in Git. The AWS account stays on the Free plan. Credits are not treated as a budget.

## What I would change in production

Per-service repositories, a signed supply chain, a non-root demo-shop security context, and a real GitOps controller pointed at a registry this cluster can pull. I would not start by giving the investigator credentials.
