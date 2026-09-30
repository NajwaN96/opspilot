# ADR 0030 — Developer portal and golden paths

## Status

Accepted

## Context

The console already shows live and simulated operations. A reviewer still has to infer ownership, the service contract, and how a new service is supposed to look. A full Backstage install would add another control plane and another set of credentials.

## Decision

OpsPilot keeps a small internal developer portal:

- `platform/catalog/catalog.json` is the service contract for storefront, checkout-api, payment-api, orders-api, and inventory-api. The file is JSON, which is valid YAML 1.2, and `platform/contract/validate.py` rejects a malformed definition.
- `templates/services/{go,node,python}` are the golden-path skeletons: health, readiness, metrics, a non-root Deployment, probes, and resource limits.
- The public Lambda serves the catalog and runbooks as GET routes. POST `/api/v1/platform/golden-path` is refused there.
- The local API may copy a template into `generated/services/<name>`. That directory is gitignored. The copy refuses an existing catalog name and a path that already exists.
- Scorecards use PASS, WARNING, and MISSING from the contract. They are not a numeric grade and not a live admission controller.

## Consequences

demo-shop security contexts are MISSING on purpose. The GitOps manifests set probes and resource limits, and they do not set `runAsNonRoot`. The templates do. CI for those services is WARNING because the monorepo pipeline builds the shared demo image and does not publish a per-service workflow.

The portal does not create GitHub repositories, clusters, or AWS resources.

## Alternatives considered

- Adopt Backstage. Rejected for this portfolio. It would duplicate the catalog and add a service that this lab does not need.
- Let the public site write generated files into the Lambda package. Rejected. The public function is read-only.
