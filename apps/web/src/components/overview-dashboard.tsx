"use client";

import Link from "next/link";
import { ErrorBlock, LoadingBlock, PageHeader, Panel } from "@/components/states";
import { ServiceTable } from "@/components/service-table";
import { StatusBadge } from "@/components/status-badge";
import { formatDuration, formatPercent } from "@/lib/format";
import { useApi } from "@/lib/use-api";
import type { Cluster, Incident, Service } from "@/lib/types";

export function OverviewDashboard() {
  const clusters = useApi<Cluster[]>("/api/v1/clusters", 4000);
  const services = useApi<Service[]>("/api/v1/services", 4000);
  const incidents = useApi<Incident[]>("/api/v1/incidents", 4000);
  const loading = (clusters.loading || services.loading || incidents.loading) && !clusters.data && !services.data;
  const error = clusters.error ?? services.error ?? incidents.error;
  const cluster = clusters.data?.[0];
  const active = incidents.data?.filter((incident) => incident.status !== "resolved") ?? [];

  if (loading) return <LoadingBlock />;
  if (!cluster || !services.data || !incidents.data) {
    return <ErrorBlock message={error ?? "Control plane unavailable"} onRetry={() => void clusters.reload()} />;
  }

  const health = cluster.health;
  return (
    <div>
      <PageHeader
        kicker={cluster.name}
        title="Overview"
        description="Live view of the simulated production cluster. Signals come from the in-memory control plane, not from a Kubernetes API."
      />
      {active.length > 0 ? (
        <div className="mb-3 border border-red-400/40 bg-red-400/10">
          {active.map((incident) => (
            <Link
              key={incident.id}
              href={`/incidents/${incident.id}`}
              className="flex flex-wrap items-center gap-x-3 gap-y-1 px-3 py-2 text-sm hover:bg-red-400/10"
            >
              <span className="font-mono text-red-200">{incident.id}</span>
              <StatusBadge value={incident.severity} />
              <span className="font-medium">{incident.title}</span>
              <span className="text-muted-foreground">{incident.serviceName}</span>
              <span className="ml-auto font-mono text-xs text-muted-foreground">{formatDuration(incident.durationSec * 1000)}</span>
            </Link>
          ))}
        </div>
      ) : (
        <div className="mb-3 border border-emerald-400/30 bg-emerald-400/10 px-3 py-2 text-sm text-emerald-200">
          No active incidents. {cluster.name} is clear.
        </div>
      )}
      <div className="mb-3 grid gap-3 sm:grid-cols-2 xl:grid-cols-5">
        <Stat label="Cluster health" value={health.state} hint={cluster.kubernetesVersion} emphasize={health.state} />
        <Stat label="Active incidents" value={String(health.activeIncidents)} hint={health.activeIncidents === 1 ? "SEV-2 open" : "Open right now"} />
        <Stat
          label="SLO compliance"
          value={`${health.servicesWithinSLO}/${health.servicesTotal}`}
          hint={`${formatPercent(health.sloCompliance)} average`}
        />
        <Stat label="Services" value={`${health.servicesHealthy}/${health.servicesTotal}`} hint="Healthy services" />
        <Stat label="Deployments today" value={String(health.deploymentsToday)} hint="Last 24 hours" />
      </div>
      <div className="grid gap-3 xl:grid-cols-[minmax(0,1.7fr)_minmax(280px,0.8fr)]">
        <Panel title="Service health" padded={false}>
          <ServiceTable services={services.data} />
        </Panel>
        <Panel title="Active incidents" padded={false}>
          {active.length === 0 ? (
            <p className="px-3 py-4 text-sm text-muted-foreground">Nothing is paging.</p>
          ) : (
            <ul>
              {active.map((incident) => (
                <li key={incident.id} className="border-b border-border px-3 py-3 last:border-b-0">
                  <div className="flex items-center gap-2">
                    <Link href={`/incidents/${incident.id}`} className="font-mono text-sm hover:underline">
                      {incident.id}
                    </Link>
                    <StatusBadge value={incident.severity} />
                    <StatusBadge value={incident.status} />
                  </div>
                  <Link href={`/incidents/${incident.id}`} className="mt-1 block text-sm font-medium hover:underline">
                    {incident.title}
                  </Link>
                  <p className="mt-1 text-xs text-muted-foreground">
                    {incident.serviceName} · {formatDuration(incident.durationSec * 1000)}
                  </p>
                </li>
              ))}
            </ul>
          )}
        </Panel>
      </div>
    </div>
  );
}

function Stat({ label, value, hint, emphasize }: { label: string; value: string; hint: string; emphasize?: string }) {
  const tone =
    emphasize === "critical"
      ? "text-red-300"
      : emphasize === "degraded"
        ? "text-amber-200"
        : emphasize === "healthy"
          ? "text-emerald-300"
          : "";
  return (
    <div className="border border-border bg-card px-3 py-2.5">
      <div className="text-[11px] uppercase tracking-wide text-muted-foreground">{label}</div>
      <div className={`mt-1 font-mono text-xl capitalize tabular-nums ${tone}`}>{value}</div>
      <div className="mt-1 text-xs text-muted-foreground">{hint}</div>
    </div>
  );
}
