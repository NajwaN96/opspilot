# ADR 0022 — OpenTofu layout and local state

## Status

Accepted

## Context

The cloud stack should be reviewable in this monorepo and should not create a state bucket before the portfolio cluster exists.

## Decision

OpenTofu 1.12.6 is the implementation. The root module is `infra/terraform/environments/dev`. It calls four modules: `network`, `iam`, `eks`, and `ecr`. The backend is local, at `infra/terraform/environments/dev/terraform.tfstate`. State files, `.terraform/`, and `dev.auto.tfvars` are gitignored.

A later remote backend can be an S3 bucket and a DynamoDB lock table created by a separate bootstrap, then selected with `tofu init -migrate-state`. That bootstrap is not part of this stack, so applying the cluster does not depend on a bucket that the same stack would have to create first.

`cloud-plan` runs fmt, init, validate, and plan. `cloud-apply` applies only an existing plan file after the account, region, environment, and `OPSPILOT_CLOUD_APPROVED=yes` checks.

## Consequences

State lives on the operator machine until someone deliberately migrates it. Losing the state file means the next apply cannot be trusted to update the existing cluster. Treat the state file as private and do not commit it.
