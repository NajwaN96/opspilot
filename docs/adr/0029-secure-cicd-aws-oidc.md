# ADR 0029 — Secure CI/CD with AWS OIDC

## Status

Accepted

## Context

The public portfolio is one Lambda function URL, `opspilot-portfolio`, in us-east-1. The EKS design under `infra/terraform` is plan-only. The local k3d cluster `opspilot-dev` is a different system and is not reachable from a GitHub-hosted runner.

A deploy job needs permission to replace that function's zip and to read whether the account is still on the Free plan. A long-lived access key in GitHub would outlive the deploy, would be copyable, and would be one leaked secret away from whatever that key could do. This account also must not be upgraded to Paid, and CI must not gain a path that applies the EKS stack.

## Decision

GitHub Actions authenticates to AWS with OIDC.

```
GitHub
  → CI quality gates
  → security gates
  → build
  → OIDC
  → AWS STS
  → least-privilege role opspilot-github-deployer
  → Lambda opspilot-portfolio
  → smoke tests
  → rollback to the previous zip when verification fails
```

Static `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` values are prohibited in workflows and in GitHub secrets. The trust policy uses `StringEquals` on three claims: audience `sts.amazonaws.com`, the GitHub subject, and ref `refs/heads/main`. There is no `Principal: "*"` and no wildcard repository.

GitHub includes the owner id and repository id in the OIDC subject (`repo:OWNER@OWNER_ID/NAME@REPO_ID:environment:aws-portfolio`). A name-only subject (`repo:OWNER/NAME:environment:aws-portfolio`) is not what Actions sends, and `sts:AssumeRoleWithWebIdentity` rejects it. The bootstrap reads `sub_claim_prefix` from the repository and refuses to continue if immutable subjects are turned off. The repository is not opted out of that claim.

The role's inline policy allows:

- `lambda:GetFunction`
- `lambda:GetFunctionConfiguration`
- `lambda:UpdateFunctionCode`
- `lambda:GetFunctionUrlConfig`

on `arn:aws:lambda:us-east-1:*:function:opspilot-portfolio` only, plus `freetier:GetAccountPlanState` on `*` so the job can refuse to continue when the plan is not FREE and ACTIVE. It does not allow EKS, EC2, RDS, S3, CloudFormation, IAM administration, Organizations, or load balancers.

CI validates `infra/terraform` with `tofu fmt`, `tofu init -backend=false`, and `tofu validate`. It never runs apply. A policy test fails if a workflow contains an apply command. `infra/aws-zero-cost` is the deployable stack, and the GitHub job does not apply that stack either. It only replaces the code of the function that already exists.

Rollback keeps the previous zip returned by `lambda:GetFunction` and uploads it again if smoke tests fail. The pipeline does not publish numbered Lambda versions, so retention does not grow without a bound.

`GET /api/v1/release` exposes version, git commit, build time, and environment. Account ids, caller ARNs, credentials, and filesystem paths are dropped.

The local lab stays `LOCAL — LIVE`. The public function stays `AWS — PORTFOLIO DEMO`. EKS stays `AWS — PLAN ONLY`.

## Consequences

The deployer role cannot be created until `OPSPILOT_GITHUB_REPOSITORY` is one real `owner/name` and GitHub returns that repository's immutable subject prefix. Guessing a name, or trusting the name without the ids, would let a later occupant of that name assume the role. `scripts/aws-oidc-bootstrap.sh` refuses a wildcard and does not print the account id.

GitHub environment `aws-portfolio` is declared on the deploy job. Required reviewers are a repository setting. They are not assumed to exist on a free private repository. Protection still comes from the main-branch trust, the CI gates, and the role scope.

One production deploy runs at a time under the concurrency group `opspilot-aws-production`.

## Alternatives considered

- Long-lived IAM user keys stored as GitHub secrets. Rejected. They do not expire with the job and are the credential class this phase removes.
- Letting CI run `tofu apply` for the portfolio stack. Rejected. Apply can create IAM and log resources. Code update cannot.
- Letting CI apply the EKS stack when credits remain. Rejected. Credits are not a budget, and EKS is plan-only.
- Publishing a Lambda version on every deploy. Rejected. Unbounded versions are unnecessary when the previous zip is enough to roll back.
