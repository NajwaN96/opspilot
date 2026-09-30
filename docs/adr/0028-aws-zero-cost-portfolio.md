# ADR 0028 — AWS zero-cost portfolio deployment

## Status

Accepted

## Context

OpsPilot already has three different things that are easy to confuse:

- a local k3d cluster, `opspilot-dev`, which runs the real control plane, PostgreSQL, demo-shop, Prometheus, OpenTelemetry, Jaeger, Grafana, and Alertmanager
- an OpenTofu EKS design under `infra/terraform` that is plan-only because EKS, a node, and public IPv4 are billable
- a need for a public AWS URL that does not upgrade the account and does not spend the $100 free-plan credits as a budget

The account in use is a Free plan. Official AWS documentation for that plan says charges are not incurred unless the account is upgraded or a paid-only service is activated. Credits still cover usage outside always-free allowances, and this project treats those credits as a safety buffer.

Official pages used to choose the stack:

- Lambda pricing: 1,000,000 requests and 400,000 GB-seconds each month are free, on the Free plan and the Paid plan.
- Lambda pricing: data transferred out of a function is charged at EC2 data-transfer rates.
- EC2 on-demand pricing: 100 GB of data transfer out to the internet is free each month, aggregated across AWS services and regions except China and GovCloud.
- CloudWatch pricing: 5 GB of log ingestion, archive storage, and Logs Insights scanning is free.
- The AWS free serverless page: API Gateway, Amazon S3, and Amplify Hosting are described as credit-backed on this plan, not as an always-free monthly allowance that starts at zero dollars of credit use.
- The AWS free web-apps page: the CloudFront flat-rate free bundle is described under a Paid plan.
- Account Management docs: Lambda, CloudWatch Logs, and IAM are in the Free plan service list. Creating them does not require a Paid plan upgrade. EKS is also listed, but it is still forbidden here because it is not free.

Expected portfolio usage is a small static site and a read-only JSON API: well under 1 GB of transfer, well under 1,000,000 requests, and well under 400,000 GB-seconds at 128 MB. Platform logs are kept for one day with application logging at FATAL. That is inside the 5 GB CloudWatch allowance. No VPC, no public IPv4 address, no provisioned concurrency, and no ephemeral storage above the included 512 MB.

## Decision

Deploy one separate stack, `infra/aws-zero-cost/`:

```
Internet
    |
    v
Lambda function URL (HTTPS, auth NONE)
    |
    +-- static OpsPilot console, labeled AWS — PORTFOLIO DEMO
    +-- read-only sanitized JSON
```

The function refuses every method except GET and HEAD. Remediation, experiments, canary controls, and investigator runs stay on the local lab. The browser bundle contains no AWS credentials, kubeconfig, database URL, or model key.

Do not apply `infra/terraform`. EKS remains `AWS — PLAN ONLY`.

Do not create S3, CloudFront, Amplify, API Gateway, DynamoDB, EC2, NAT, a load balancer, RDS, or a public IPv4 address. Those were rejected because they are either credit-backed, paid-plan features, or unused. `OPSPILOT_PAID_CLOUD_OVERRIDE` stays unset. `OPSPILOT_ZERO_COST_DEPLOYMENT=yes` is required and does not waive the allowlist.

The local console is unchanged when `NEXT_PUBLIC_OPSPILOT_RUNTIME` is unset. The AWS build sets it to `aws-portfolio-demo`, stops polling, and hides approval buttons.

## Consequences

The public URL is a real AWS deployment of the console and a demo API. It is not a live view of `opspilot-dev` and it is not an EKS cluster. Out-of-pocket charges are not expected while the account remains on the Free plan and usage stays inside the allowances above. That is not the same statement as "the AWS bill API returned $0": Cost Explorer is not enabled, and enabling it is not required for this path. `scripts/aws-cost-status.sh` reads the free-plan API and the portfolio resources. Built-in credit emails are not spending caps.

If a later change adds a resource outside the allowlist, `scripts/aws-zero-cost-preflight.sh` fails closed.
