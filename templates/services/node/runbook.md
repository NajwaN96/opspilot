# __SERVICE_NAME__

## Symptoms
Error rate or latency outside the SLO.

## Detection
Prometheus records the service. OpsPilot opens an incident from its own rules. The browser does not send PromQL.

## Evidence
Metrics, traces, and the Deployment image.

## Likely causes
A bad release or a dependency named in service.yaml.

## Safe investigation
Read the incident. Do not exec into the pod from the console.

## Allowed remediation
Only an allowlisted action for this service, after a person approves it.

## Verification
The SLO window recovers and the audit trail records the result.

## Escalation
If the action is not on the allowlist, leave the executor idle.
