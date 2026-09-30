"use client";

import { StatusBadge } from "@/components/status-badge";
import { PageHeader, Panel } from "@/components/states";
import { isPortfolioRuntime } from "@/lib/runtime";
import { useApi } from "@/lib/use-api";
import type { Health, ReleaseInfo } from "@/lib/types";

const localRows = [
  ["Data source", "In-memory repository. PostgreSQL is the planned replacement."],
  ["Cluster access", "None. The executor is simulated and does not load a kubeconfig."],
  ["Remediation", "INC-142 stays simulated. A detected incident can approve one real rollback of demo-shop/payment-api from 1.5.0-bad to 1.4.2."],
  ["Policy", "The live action cannot target another namespace or Deployment, and the browser cannot supply an image."],
  ["AI investigator", "Server-side only. It reads a prepared evidence snapshot and cannot execute. The API key is not shown here."],
  ["Telemetry", "Metrics, logs, traces, and events on the incident page are simulated."],
];

const portfolioRows = [
  ["Deployment", "AWS — PORTFOLIO DEMO. A Lambda function URL serves this console and a read-only sample API."],
  ["Kubernetes", "Not connected. The live lab remains local k3d. EKS stays plan-only and is not applied."],
  ["Remediation", "Disabled. POST routes return a refusal. The constrained executor is not deployed here."],
  ["Data", "Deterministic sanitized samples. They are not live Prometheus, Jaeger, PostgreSQL, or kubeconfig output."],
  ["Secrets", "No AWS credentials, OpenAI key, database URL, or kubeconfig are bundled in this site."],
];

export function SettingsPage() {
  const health = useApi<Health>("/health", 10000);
  const release = useApi<ReleaseInfo>("/api/v1/release");
  const portfolio = isPortfolioRuntime();
  const rows = portfolio ? portfolioRows : localRows;
  return (
    <div>
      <PageHeader
        kicker={portfolio ? "AWS — PORTFOLIO DEMO" : "Local MVP"}
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
        <div className="grid content-start gap-3">
        <Panel title="Release">
          {release.data ? (
            <dl className="space-y-2 text-sm">
              <Field label="Environment" value={release.data.environment} />
              <Field label="Version" value={release.data.version} />
              <Field label="Git commit" value={release.data.gitCommit} />
              <Field label="Built" value={release.data.buildTime} />
            </dl>
          ) : (
            <p className="text-sm text-muted-foreground">{release.error ?? "Reading release metadata"}</p>
          )}
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
    </div>
  );
}

function Field({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt className="text-[11px] uppercase tracking-wide text-muted-foreground">{label}</dt>
      <dd className="font-mono">{value}</dd>
    </div>
  );
}
