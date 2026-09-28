# Local infrastructure

The supported way to run OpsPilot is two processes: `go run` in `apps/api` and `npm run dev` in `apps/web`. See the repository README.

`docker-compose.yml` is an optional packaging of those same processes. It does not add a database or a cluster. The images are not required for development, and this environment may not have a Docker daemon.

From the repository root, if Docker is available:

```bash
docker compose -f infra/docker-compose.yml up --build
```

- Console: http://127.0.0.1:3461
- API: http://127.0.0.1:8094

The compose file sets `API_PROXY_URL=http://api:8094` so the Next.js server proxies to the API container. Incident state is still in memory and disappears when the API container stops.
