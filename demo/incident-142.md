# Demo: INC-142

This is the script for the simulated payment incident. The API process must be freshly started so the incident is still investigating.

1. Open the console overview. `production-01` is critical. One active incident is listed: **INC-142**, SEV-2, `payment-api`.
2. The service table shows `payment-api` at 98.71% availability, 1.8s p95, and 12.4% errors. The other four services are healthy.
3. Open **INC-142**.
4. Read the timeline from 14:31 (deploy of `v1.8.2`) through 14:34 (incident opened).
5. Open evidence:
   - Metrics show the latency and error jump after 14:32.
   - Logs include `database connection pool exhausted`.
   - The trace is `checkout-api` → `payment-api` → `postgres`, with the database span marked slow.
   - Kubernetes events show the v1.8.2 rollout and a readiness warning.
6. The root cause panel states: database connection leak introduced by `payment-api:v1.8.2`, confidence 91%. The panel is labeled simulated.
7. The remediation panel recommends rollback from `v1.8.2` to `v1.8.1`, risk LOW, policy Allowed.
8. Choose **Approve & Execute**.
9. The steps advance: approval recorded, rollback started, deployment progressing, health verification, incident resolved.
10. The header metrics move to about 0.3% errors, 240ms p95, and 37% database connections. Status becomes Resolved. The service version becomes `v1.8.1`.

No pod is changed. Restarting the API restores the incident.
