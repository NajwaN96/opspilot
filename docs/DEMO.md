# Demo script

About eight minutes. Say what is live before you click.

## Open

OpsPilot is a Kubernetes reliability control plane. The laptop cluster is live. The public URL is a read-only AWS portfolio. The EKS stack is plan only and has not been applied.

## Architecture

Open Architecture. Walk the three columns: Lambda and OIDC, k3d through approval to verification, OpenTofu marked plan only.

## Catalog

Open Developer Portal. Show payment-api: owner, canary strategy, SLO, and the scorecard. Security context is MISSING on the running demo and present on the golden-path template. That is intentional.

Open Golden path. Generate a preview for a new Go service. On AWS, stop at the preview. Locally, the write goes to `generated/services` and will not replace payment-api.

## SLO and a bad release

On the local lab, show payment-api healthy at `opspilot-demo:1.4.2`. From Reliability Lab, deploy the controlled bad release. Wait for Prometheus to move the error rate. An incident opens from the detector, not from the button alone.

## Incident

Open the incident. The lifecycle runs from detected through approval to resolved. Evidence shows the Prometheus window, the trace into `charge`, and the image. The AI panel either cites the snapshot or says AI Investigator Unavailable. Detection still stands. Approve the rollback. The executor changes only that Deployment. Prometheus recovers, and the audit row is on the incident.

Reset Demo returns the baseline. Do not leave the bad image running.

## Progressive delivery

Show a good canary promoting through 5, 25, 50, and 100 with the SLO gate. Show a bad canary failing the gate and aborting back to 1.4.2. The stable Deployment is the one that stays.

## CI and AWS

Open the GitHub Actions run for `main`. The gates include tests, scans, both OpenTofu stacks, and the zero-cost policy. The deploy job assumes `opspilot-github-deployer` and updates the Lambda. Settings shows the git commit. A POST to reset or remediate on the public URL returns 403.

## Close

Security page: AI read-only, public AWS read-only, remediation allowlisted, EKS plan only, no long-lived access keys.
