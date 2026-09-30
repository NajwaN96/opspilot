import Link from "next/link";
import type { Incident } from "@/lib/types";

const STEPS = [
  "Detected",
  "Investigating",
  "Diagnosed",
  "Remediation proposed",
  "Approval required",
  "Executing",
  "Verifying",
  "Resolved",
];

function indexFor(incident: Incident): number {
  const remediation = incident.remediation?.status;
  if (incident.status === "resolved" || remediation === "succeeded") return 7;
  if (remediation === "verifying") return 6;
  if (remediation === "rolling" || incident.status === "mitigating") return 5;
  if (incident.recommendation && remediation !== "rejected") return 4;
  if (incident.analysis || incident.recommendation) return 3;
  if (incident.evidence) return 2;
  if (incident.status === "investigating") return 1;
  return 0;
}

export function IncidentLifecycle({ incident }: { incident: Incident }) {
  const current = indexFor(incident);
  const runbook = incident.serviceName === "payment-api" ? "payment-api-high-error-rate" : "dependency-failure";
  return (
    <div className="mb-3 surface-card overflow-hidden rounded-xl px-3 py-2">
      <div className="mb-2 flex flex-wrap items-center justify-between gap-2">
        <div className="text-[11px] uppercase tracking-wide text-muted-foreground">Lifecycle</div>
        <Link className="text-[11px] hover:underline" href={`/runbooks/${runbook}`}>
          Runbook
        </Link>
      </div>
      <ol className="flex flex-wrap gap-1">
        {STEPS.map((step, index) => (
          <li
            key={step}
            className={`rounded-md px-1.5 py-0.5 text-[11px] ${index === current ? "bg-primary text-primary-foreground" : index < current ? "text-status-success" : "text-muted-foreground"}`}
          >
            {step}
          </li>
        ))}
      </ol>
    </div>
  );
}
