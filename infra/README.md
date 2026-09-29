# Local infrastructure

PostgreSQL is the dependency the API needs for durable approvals. The Next.js console stays on the host so the dev server can reload.

```bash
docker compose -f infra/docker-compose.yml up -d postgres
```

The compose file also defines an API service. It persists to Postgres. It does not mount a kubeconfig, so Kubernetes discovery from that container stays disconnected unless you add a read-only mount yourself. The supported path for discovery is `go run` on the host, which uses the local kubeconfig.

Development credentials are `opspilot` / `opspilot` on `127.0.0.1:5432`. They are not production secrets. Override them with `POSTGRES_USER`, `POSTGRES_PASSWORD`, and `POSTGRES_DB`.

## Cluster

```bash
infra/scripts/up-dev-cluster.sh
```

That creates k3d cluster `opspilot-dev` (one server, no agents, k3s v1.31.4) and applies `gitops/overlays/local`.

Namespaces:

- `opspilot-system` for later OpsPilot components
- `demo-shop` for storefront, checkout-api, payment-api, orders-api, and inventory-api

If the k3d load balancer cannot open the API port, the script points kubeconfig at the server container IP. Do not commit that kubeconfig.
