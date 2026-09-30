# Phase 8 — CI/CD and portfolio release

## What runs where

| Venue | Label | How it changes |
| --- | --- | --- |
| Laptop k3d `opspilot-dev` | LOCAL — LIVE | A person on the machine. GitHub runners do not connect to it. |
| Lambda `opspilot-portfolio` | AWS — PORTFOLIO DEMO | GitHub Actions on `main`, OIDC, code update only. |
| `infra/terraform` EKS | AWS — PLAN ONLY | `tofu validate` in CI. Apply is refused. |

`OPSPILOT_PAID_CLOUD_OVERRIDE` stays unset. Do not upgrade the AWS account.

## Pull request

`.github/workflows/ci.yml` runs on pull requests and on pushes that are not `main`. Jobs:

1. Repository policy and the static secret scan
2. Go fmt, vet, test, race, and build
3. Vitest
4. ESLint
5. TypeScript
6. Next.js production build and the portfolio zip, checksum, and release metadata
7. Gitleaks
8. govulncheck and npm audit
9. Trivy filesystem scan
10. OpenTofu fmt and validate for the plan-only stack and the portfolio stack
11. Kustomize render
12. Zero-cost guard tests, including the regression that fails if a workflow applies infrastructure
13. Demo image build, not pushed

No job receives `id-token: write`. No job has AWS keys.

## Production deploy

`.github/workflows/deploy-aws.yml` runs on a push to `main`. Concurrency group `opspilot-aws-production` allows one deploy. The deploy job waits for the reusable CI workflow, uses environment `aws-portfolio`, and requests an OIDC token.

Before any AWS call the job checks that variable `OPSPILOT_AWS_ROLE_ARN` names role `opspilot-github-deployer`. It does not print the ARN. `aws-actions/configure-aws-credentials` exchanges the OIDC token for temporary credentials.

`scripts/aws-portfolio-publish.sh` then:

1. Refuses to run when the paid override is set.
2. Checks account suffix `3851`, region `us-east-1`, function `opspilot-portfolio`, and Free plan ACTIVE.
3. Downloads the current function zip.
4. Uploads the new zip with `lambda:UpdateFunctionCode`.
5. Waits until the function is updated.
6. Requests the public URL.

Smoke expectations:

- HTTP 200 for `/`, `/services`, `/incidents`, `/infrastructure`, `/rollouts`, `/settings`, `/health`, `/api/v1/cloud/status`, and `/api/v1/release`
- `/api/v1/cloud/status` still says `AWS — PORTFOLIO DEMO`
- `/api/v1/release` returns only version, git commit, build time, and `aws-portfolio-demo`
- HTTP 403 for `POST /api/v1/demo/reset` and `POST /api/v1/incidents/INC-142/remediations`

If that fails, the previous zip is uploaded again, smoke runs again, and the workflow fails. The publisher does not call `publish-version`.

Release metadata in the Actions artifact records the commit, build time, SHA-256, workflow run, environment, function name, and region. It does not record credentials or the account id.

## One-time role

Run this only after the GitHub repository `owner/name` is real. Do not invent one.

```bash
export OPSPILOT_ZERO_COST_DEPLOYMENT=yes
export OPSPILOT_GITHUB_REPOSITORY=owner/name
scripts/aws-oidc-bootstrap.sh
```

Then set the GitHub Actions variable `OPSPILOT_AWS_ROLE_ARN` to the role ARN. Do not also create an access key. Optional required reviewers on environment `aws-portfolio` are configured in GitHub by an admin. The workflow does not depend on a paid ruleset.

A local package check that does not call AWS:

```bash
infra/aws-zero-cost/package.sh
scripts/aws-portfolio-publish.sh --dry-run
```

## Why CI cannot deploy EKS

The plan-only stack creates a cluster, nodes, a VPC, and public addresses. Those are billable. The deploy role cannot call EKS or EC2, and `infra/aws-zero-cost/ci_policy.py` fails the build if a workflow contains an apply command or an EKS CLI command. `infra/scripts/cost-classify` still exits non-zero for that stack while the paid override is unset.
