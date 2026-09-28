# ADR 0003 — Remediation is a proposal, a policy decision, then a constrained action

## Status

Accepted

## Context

The product will eventually let an investigator recommend a fix. If that component can also apply the fix, a bad recommendation is a cluster incident of its own. The MVP has to demonstrate the safe shape even though both the analysis and the executor are fake.

## Decision

A remediation is three separate steps.

1. **Proposal.** Data on the incident: action, from version, to version, risk, and whether policy currently allows it. The UI renders this. It does not invent an action.
2. **Authorization.** `Service.StartRemediation` loads the incident and rejects the call unless the proposal is allowed and the requested action matches it. A second call returns the existing run instead of starting another one.
3. **Execution.** The executor receives a `Request` that already names the incident, action, service, namespace, and version pair. The simulated executor logs that request and returns. It cannot select a different action.

For `INC-142` the only allowed request is `rollback` of `payment-api` from `v1.8.2` to `v1.8.1`, risk `LOW`, policy `Allowed`. Anything else, including a rollback of a resolved historical incident, is a 400 before the executor runs.

The UI copy states that the run is simulated. The API response sets `simulated: true`.

Recovery metrics are a function of how many workflow steps have completed. They are not read from a cluster.

## Consequences

A future model can only add a proposal. Wiring it to the cluster means implementing `executor.Executor` with a client that can perform that one rollback, and leaving the policy check where it is.

Human approval is the button. There is no auto-remediation flag. Adding one later should be an explicit policy change, not a shortcut in the executor.

The audit trail is in memory only. A durable approval record is required before a real executor is enabled.

## Alternatives

Letting the UI animate a rollback locally would have been shorter and would have taught the wrong boundary. The approval has to hit the API so policy is tested.

Shelling out to `kubectl` from the demo was rejected. A portfolio button that changes a cluster, even a local one, is the failure mode this design exists to prevent.
