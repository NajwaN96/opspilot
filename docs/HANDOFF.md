# Handoff

Local lab header: **LOCAL — LIVE**. Public site banner: **AWS PUBLIC PORTFOLIO — READ ONLY**. Those are different systems. The Lambda URL does not talk to `opspilot-dev`.

`INC-142` is the seeded simulation. `INC-REAL-ac2c87e5` is the resolved manual test and should be left in PostgreSQL.

## Start the local demo

Requirements are in the README: Node.js 22, npm, Go 1.27.1, Docker, k3d, kubectl.

PostgreSQL:

```bash
docker compose -f infra/docker-compose.yml up -d postgres
```

Cluster, demo-shop, Prometheus, the collector, Jaeger, Grafana, and Alertmanager:

```bash
infra/scripts/up-dev-cluster.sh
```

API, from `apps/api`, with the database URL set. Without `OPSPILOT_DATABASE_URL` the API stays on the in-memory simulation and does not run detection.

```bash
cd apps/api
export OPSPILOT_DATABASE_URL=postgres://opspilot:opspilot@127.0.0.1:5432/opspilot?sslmode=disable
export OPSPILOT_ENV=development
go run ./cmd/opspilot
```

Console, from `apps/web`. Leave this process running. It listens on port 3461.

```bash
cd apps/web
npm install
npm run dev
```

Open [http://127.0.0.1:3461](http://127.0.0.1:3461). The API is [http://127.0.0.1:8094](http://127.0.0.1:8094).

## Stop it

Stop the `go run` and `npm run dev` processes with Ctrl-C.

```bash
docker compose -f infra/docker-compose.yml stop postgres
k3d cluster stop opspilot-dev
```

`k3d cluster stop` keeps the cluster on disk. `infra/scripts/up-dev-cluster.sh` brings it back. Do not run `k3d cluster delete` unless you intend to drop the lab.

## Reset it

`POST /api/v1/demo/reset` is development-only. The console shows **Reset Demo** when the API is not in production mode. It restores the payment-api baseline image when the current image is one of the known demo tags, scales the canary to zero, closes an active canary rollout, and clears the `INC-142` simulated remediation. It does not delete `INC-REAL-…` rows and it does not delete the cluster.

```bash
curl -sS -X POST http://127.0.0.1:8094/api/v1/demo/reset -H 'content-type: application/json' -d '{}'
```

The same route on the public Lambda URL returns 403.

## Bad-release demo

Local console only. Reliability Lab, **Deploy Bad payment**. That is `POST /api/v1/rollouts/payment-api/bad` with an empty body. The API ignores any image name.

Wait until Incidents shows a new `INC-REAL-…` row. The verified example is `INC-REAL-ac2c87e5` (77 requests, 34.6% error rate, p95 975 ms, version `1.5.0-bad`). Open it and click **Approve & Execute**. That sends `{ "action": "rollback-payment-api" }`. Prometheus must recover before the status becomes `resolved`.

Do not use **Start Experiment** for this path. That fault does not change the Deployment.

Details: [runbooks/payment-api-rollback.md](../runbooks/payment-api-rollback.md).

## Canary demo

Local console, Reliability Lab:

- **Start healthy canary 1.5.0** is `POST /api/v1/lab/payment-api/canary/good`
- **Start bad canary 1.6.0-bad** is `POST /api/v1/lab/payment-api/canary/bad`

Watch Rollouts. Promotion and abort still need a person. Both routes are refused when `OPSPILOT_ENV=production`.

## Verify Kubernetes

```bash
kubectl config current-context
kubectl -n demo-shop get deploy payment-api payment-api-canary
kubectl -n demo-shop exec deploy/payment-api -- wget -qO- http://127.0.0.1:8080/
```

The context is `k3d-opspilot-dev`. A clean baseline prints image `opspilot-demo:1.4.2`, ready 1, desired 1, canary desired 0, and `{"service":"payment-api","version":"1.4.2"}`.

## Verify observability

```bash
curl -sS http://127.0.0.1:8094/health
curl -sS http://127.0.0.1:8094/ready
kubectl -n opspilot-system get deploy prometheus otel-collector jaeger grafana alertmanager
```

`/ready` should report database `ok` and kubernetes, prometheus, opentelemetry, traces, grafana, and alertmanager `connected`. `ai` may be `unauthorized`. That does not stop detection.

## Grafana

Anonymous Viewer. From the repo README, the host port used in development is 3481:

```bash
kubectl -n opspilot-system port-forward svc/grafana 3481:3000
```

Open [http://127.0.0.1:3481](http://127.0.0.1:3481). The payment-api dashboard is provisioned from `gitops/base/observability.yaml`. Grafana cannot change a rollout.

## Jaeger

The Service port in the manifest is 16686:

```bash
kubectl -n opspilot-system port-forward svc/jaeger 16686:16686
```

Open [http://127.0.0.1:16686](http://127.0.0.1:16686) and look up service `payment-api`.

Alertmanager, when you need it, matches the README host port:

```bash
kubectl -n opspilot-system port-forward svc/alertmanager 3482:9093
```

## Public AWS portfolio

[https://jcqljkrf25oijz4cryqw7c76lq0wmzhi.lambda-url.us-east-1.on.aws/](https://jcqljkrf25oijz4cryqw7c76lq0wmzhi.lambda-url.us-east-1.on.aws/)

Label it **AWS PUBLIC PORTFOLIO — READ ONLY**. `GET /api/v1/release` shows the deployed git commit. POST, including demo reset and remediation, returns 403.

## How a deploy reaches AWS

A push to `main` runs `.github/workflows/deploy-aws.yml`. The job assumes role `opspilot-github-deployer` with a GitHub OIDC token and updates the existing Lambda function `opspilot-portfolio`. It does not use a long-lived AWS access key. It does not apply `infra/terraform`. `OPSPILOT_PAID_CLOUD_OVERRIDE` stays unset.

## Frontend process

`npm run dev` is `next dev -p 3461 -H 0.0.0.0` in `apps/web`. If the browser cannot open [http://127.0.0.1:3461](http://127.0.0.1:3461), the Next process has exited. Start it again from `apps/web` and leave it in the foreground or in its own terminal. Confirm with:

```bash
curl -sS -o /dev/null -w '%{http_code}\n' http://127.0.0.1:3461/reliability-lab
```

`200` means the console is up. A connection error means it is not listening. Do not restart the cluster to fix that.

The console proxies `/api` to `127.0.0.1:8094`. If pages load and data calls fail, check the API first.

## Return payment-api to 1.4.2

If the image is `opspilot-demo:1.5.0-bad` and an `INC-REAL-…` incident is open, use **Approve & Execute** on that incident. That is the only supported rollback.

If the bad release is up and you need the baseline without walking the incident, **Reset Demo** calls the same image restore for a known demo tag and does not delete resolved `INC-REAL-…` rows. Prefer the approval path when you are demonstrating the incident.

Check the result:

```bash
kubectl -n demo-shop get deploy payment-api -o jsonpath='{.spec.template.spec.containers[0].image}{"\n"}'
```

The clean value is `opspilot-demo:1.4.2`.
