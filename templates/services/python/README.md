# __SERVICE_NAME__

Golden path skeleton for a python service. OpsPilot fills `__SERVICE_NAME__`, `__OWNER__`, `__SLO__`, and `__STRATEGY__` when a local preview is written under `generated/services/`. The public AWS site does not write this tree.

Health: `GET /health` and `GET /ready`. Metrics: `GET /metrics`. Traces export OTLP when `OTEL_EXPORTER_OTLP_ENDPOINT` is set.
