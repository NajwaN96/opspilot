# Platform boundary

OpsPilot does not ship cluster manifests yet. There is nothing here to `kubectl apply`.

The action boundary lives in `apps/api/internal/executor`.

```go
type Executor interface {
    Execute(ctx context.Context, req Request) error
}
```

`Request` names one incident, one action, one service, one namespace, and a from/to version. The service layer fills that struct only after policy allows it.

`Simulated` is the implementation in this MVP. A later Kubernetes implementation belongs in this package and must:

- use a client scoped to that one action
- refuse any request the service did not already authorize
- return an error rather than retry with a broader permission
- leave verification to a read-only check

Collection (metrics, logs, traces, events) is a different boundary and does not belong on the executor.
