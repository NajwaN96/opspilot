"use client";

import { StatusBadge } from "@/components/status-badge";
import { PageHeader, Panel } from "@/components/states";
import { useApi } from "@/lib/use-api";
import type { Health } from "@/lib/types";

const rows = [
  ["Data source", "In-memory repository. PostgreSQL is the planned replacement."],
  ["Cluster access", "None. The executor is simulated and does not load a kubeconfig."],
  ["Remediation", "INC-142 stays simulated. A detected incident can approve one real rollback of demo-shop/payment-api from 1.5.0-bad to 1.4.2."],
  ["Policy", "The live action cannot target another namespace or Deployment, and the browser cannot supply an image."],
  ["AI investigator", "Server-side only. It reads a prepared evidence snapshot and cannot execute. The API key is not shown here."],
  ["Telemetry", "Metrics, logs, traces, and events on the incident page are simulated."],
];

export function SettingsPage() {
  const health = useApi<Health>("/health", 10000);
  return (
    <div>
      <PageHeader
        kicker="Local MVP"
        title="Settings"
        description="Read-only description of this environment. There is nothing to save, and no secrets are stored here."
      />
      <div className="grid gap-3 lg:grid-cols-[minmax(0,1fr)_280px]">
        <Panel title="Control plane" padded={false}>
          <dl>
            {rows.map(([label, value]) => (
              <div key={label} className="grid gap-1 border-b border-border px-3 py-3 sm:grid-cols-[180px_minmax(0,1fr)]">
                <dt className="text-xs uppercase tracking-wide text-muted-foreground">{label}</dt>
                <dd className="text-sm">{value}</dd>
              </div>
            ))}
          </dl>
        </Panel>
        <Panel title="API">
          <div className="flex items-center justify-between">
            <span className="text-sm">Health</span>
            {health.data ? <StatusBadge value={health.data.status === "ok" ? "healthy" : health.data.status} /> : <span className="text-xs text-muted-foreground">{health.error ?? "Checking"}</span>}
          </div>
          <dl className="mt-3 space-y-2 text-sm">
            <div>
              <dt className="text-[11px] uppercase tracking-wide text-muted-foreground">Service</dt>
              <dd className="font-mono">{health.data?.service ?? "opspilot-api"}</dd>
            </div>
            <div>
              <dt className="text-[11px] uppercase tracking-wide text-muted-foreground">Version</dt>
              <dd className="font-mono">{health.data?.version ?? "—"}</dd>
            </div>
            <div>
              <dt className="text-[11px] uppercase tracking-wide text-muted-foreground">Browser path</dt>
              <dd className="font-mono">/api/v1</dd>
            </div>
          </dl>
        </Panel>
      </div>
    </div>
  );
}
