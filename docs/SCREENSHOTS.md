# Screenshot plan

Use these names under `docs/images/` for the GitHub portfolio. Capture the local console at `http://127.0.0.1:3461` unless the row says public. Do not photograph credentials, account ids, tokens, authorization headers, or `.env`.

The three older files `overview.webp`, `incident.webp`, and `incident-resolved.webp` are the seeded `INC-142` simulation. Do not relabel them as `INC-REAL-ac2c87e5`.

| File | Page | What must be visible |
| --- | --- | --- |
| `01-overview` | `/` | **LOCAL — LIVE**, `opspilot-dev`, active services |
| `02-developer-portal` | `/developer-portal` | Catalog and scorecard, **LOCAL — LIVE** |
| `03-service-catalog` | `/developer-portal/services/payment-api` | Owner, SLO, canary strategy, version 1.4.2 |
| `04-payment-api` | `/services/k8s_demo-shop_payment-api` | Image `opspilot-demo:1.4.2`, Ready, Prometheus panel |
| `05-slos` | `/slos` | payment-api SLO from the local window, not a fake invoice |
| `06-reliability-lab` | `/reliability-lab` | **Deploy Bad payment** visible and not clicked when the lab is clean |
| `07-real-incident-detection` | `/incidents/INC-REAL-ac2c87e5` | Id, rule `PAYMENT_API_RELIABILITY_DEGRADATION`, status path |
| `08-incident-evidence` | same incident, evidence section | 77 requests, 34.6% error rate, p95 975 ms, version `1.5.0-bad` |
| `09-human-approval` | same incident, approval section | `rollback-payment-api`, human approval, no extra actions |
| `10-rollback-verification` | same incident, verification | Image back to `1.4.2`, Prometheus recovery |
| `11-resolved-incident` | same incident, header | Status `resolved` |
| `12-rollouts` | `/rollouts` | Canary stages. No active rollout on a clean baseline |
| `13-architecture` | `/architecture` | Three venues, EKS marked plan only |
| `14-security` | `/security` | Read-only AI, allowlisted rollback, public POST blocked |
| `15-github-actions` | GitHub Actions for `main` | Green `deploy-aws` run. Crop any token or account id |

Public pages for 02, 13, and 14 may be shot on the Lambda URL. Label those shots **AWS PUBLIC PORTFOLIO — READ ONLY**. Do not mix a public banner into a local incident shot.

`15-github-actions` needs a logged-in GitHub view. Capture it manually if automation cannot open the private Actions UI.

## Captured from the local console

These PNG files were taken from `http://127.0.0.1:3461` after the verified run, with payment-api on `1.4.2` and `INC-REAL-ac2c87e5` resolved. No control was clicked.

| File | Result |
| --- | --- |
| `01-overview.png` | **LOCAL — LIVE**. The top banner is seeded `INC-142`. The payment-api telemetry row on that page is the live Prometheus window |
| `02-developer-portal.png` | Developer portal |
| `03-service-catalog.png` | payment-api contract, version 1.4.2, security context MISSING |
| `04-payment-api.png` | Kubernetes record `opspilot-demo:1.4.2`, Ready 1/1. This view does not repeat the live error-rate chart; that chart is on the overview and on the incident |
| `05-slos.png` | SLO page |
| `06-reliability-lab.png` | Clean lab. Past experiments are stopped. **Deploy Bad payment** was not clicked |
| `07-real-incident-detection.png` | Full resolved incident |
| `08-incident-evidence.png` | Same full page. Metrics evidence is 77 requests, 34.6%, p95 975 ms |
| `09-human-approval.png` | Same full page. Approval recorded, AI unauthorized |
| `10-rollback-verification.png` | Same full page. Verification succeeded on `opspilot-demo:1.4.2` |
| `11-resolved-incident.png` | Same full page. Status resolved |
| `12-rollouts.png` | Historical canaries. Live canary traffic 0%. The selected rollout links `INC-REAL-ac2c87e5` resolved |
| `13-architecture.png` | Architecture |
| `14-security.png` | Boundaries, EKS plan only, paid override disabled |

`08` through `11` are the same full-page capture as `07`. The resolved incident shows detection, evidence, approval, verification, and resolution together. They are not separate staged moments and they are not drawings.

## Still manual

`15-github-actions` was not captured. The Actions UI needs a signed-in GitHub session, and a shot must be cropped so it does not include tokens or an account id. Take it from the latest green `deploy-aws` run on `main` after this closeout push.
