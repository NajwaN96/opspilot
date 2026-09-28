"use client";

import Link from "next/link";
import { ErrorBlock, LoadingBlock, PageHeader, Panel } from "@/components/states";
import { ServiceTable } from "@/components/service-table";
import { StatusBadge } from "@/components/status-badge";
import { formatAgo, formatDuration, formatLatency, formatPercent } from "@/lib/format";
import { useApi } from "@/lib/use-api";
import type { Cluster, Incident, KubernetesStatus, Service, TelemetrySnapshot, TelemetryStatus, Workload } from "@/lib/types";

export function OverviewDashboard() {
  const clusters = useApi<Cluster[]>("/api/v1/clusters", 4000);
  const services = useApi<Service[]>("/api/v1/services", 4000);
  const incidents = useApi<Incident[]>("/api/v1/incidents", 4000);
  const kubernetes = useApi<KubernetesStatus>("/api/v1/kubernetes/status", 4000);
  const workloads = useApi<Workload[]>("/api/v1/kubernetes/workloads", 4000);
  const telemetry = useApi<TelemetryStatus>("/api/v1/telemetry/status", 5000);
  const payment = useApi<TelemetrySnapshot>("/api/v1/services/k8s_demo-shop_payment-api/telemetry", 5000);
  const loading = (clusters.loading || services.loading || incidents.loading) && !clusters.data && !services.data;
  const error = clusters.error ?? services.error ?? incidents.error;
  const cluster = clusters.data?.find((item) => item.simulated) ?? clusters.data?.[0];
  const simulatedServices = services.data?.filter((service) => service.source !== "kubernetes") ?? [];
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
        description="Simulated production-01 stays separate from demo-shop. Kubernetes health and payment-api telemetry below come from the local cluster."
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
              <span className="text-[11px] uppercase tracking-wide text-muted-foreground">
                {incident.origin === "detection-engine" ? "Detection Engine" : "Simulation"}
              </span>
              <span className="ml-auto font-mono text-xs text-muted-foreground">{formatDuration(incident.durationSec * 1000)}</span>
            </Link>
          ))}
        </div>
      ) : (
        <div className="mb-3 border border-emerald-400/30 bg-emerald-400/10 px-3 py-2 text-sm text-emerald-200">
          No active incidents. {cluster.name} is clear.
        </div>
      )}
      <EnvironmentStrip status={kubernetes.data} telemetry={telemetry.data} payment={payment.data} />
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
        <Panel title="Simulated service health" action={<span className="text-[11px] text-muted-foreground">Source: Simulation</span>} padded={false}>
          <ServiceTable services={simulatedServices} />
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
      <div className="mt-3">
        <Panel
          title="Kubernetes workloads"
          action={<span className="text-[11px] text-muted-foreground">Source: Kubernetes · {kubernetes.data?.namespace ?? "demo-shop"}</span>}
          padded={false}
        >
          <WorkloadTable workloads={workloads.data} connectivity={kubernetes.data?.connectivity} message={workloads.error ?? kubernetes.data?.message} />
        </Panel>
      </div>
    </div>
  );
}

function EnvironmentStrip({
  status,
  telemetry,
  payment,
}: {
  status?: KubernetesStatus | null;
  telemetry?: TelemetryStatus | null;
  payment?: TelemetrySnapshot | null;
}) {
  return (
    <div className="mb-3 space-y-2">
      <div className="grid gap-2 border border-border bg-card px-3 py-2 text-sm sm:grid-cols-4">
        <Field label="Environment" value="LOCAL" />
        <Field label="Cluster" value={status?.cluster ?? "opspilot-dev"} />
        <div>
          <div className="text-[11px] uppercase tracking-wide text-muted-foreground">Kubernetes</div>
          <div className="mt-1">
            <StatusBadge value={status?.connectivity ?? "disconnected"} />
          </div>
        </div>
        <Field label="Namespace" value={status?.namespace ?? "demo-shop"} />
      </div>
      <div className="grid gap-2 border border-border bg-card px-3 py-2 text-sm sm:grid-cols-2 xl:grid-cols-4">
        <Connected label="Prometheus" value={telemetry?.prometheus} />
        <Connected label="OpenTelemetry" value={telemetry?.opentelemetry} />
        <Connected label="Trace backend" value={telemetry?.traces} />
        <Field label="payment-api" value={paymentLine(payment)} />
      </div>
    </div>
  );
}

function paymentLine(payment?: TelemetrySnapshot | null): string {
  if (!payment?.available) return "Telemetry unavailable";
  return `${payment.requestRate.toFixed(2)} req/s · ${formatPercent(payment.errorRate * 100)} errors · p95 ${formatLatency(payment.p95LatencyMs)}`;
}

function Connected({ label, value }: { label: string; value?: string }) {
  const connected = value === "connected";
  return (
    <div>
      <div className="text-[11px] uppercase tracking-wide text-muted-foreground">{label}</div>
      <div className={`mt-1 text-sm ${connected ? "text-emerald-300" : "text-amber-200"}`}>{connected ? "Connected" : "Unavailable"}</div>
    </div>
  );
}

function Field({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <div className="text-[11px] uppercase tracking-wide text-muted-foreground">{label}</div>
      <div className="mt-1 font-mono text-sm">{value}</div>
    </div>
  );
}

function WorkloadTable({ workloads, connectivity, message }: { workloads?: Workload[] | null; connectivity?: string; message?: string }) {
  if (connectivity === "disconnected") {
    return <p className="px-3 py-4 text-sm text-muted-foreground">{message || "Local cluster is disconnected. No workloads are invented for this view."}</p>;
  }
  if (!workloads) {
    return <p className="px-3 py-4 text-sm text-muted-foreground">Waiting for the Kubernetes read.</p>;
  }
  if (workloads.length === 0) {
    return <p className="px-3 py-4 text-sm text-muted-foreground">No Deployments in the discovery namespace.</p>;
  }
  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-border text-left text-[11px] uppercase tracking-wide text-muted-foreground">
            {["Workload", "Namespace", "Version", "Desired", "Ready", "Restarts", "Status", "Last observed"].map((heading) => (
              <th key={heading} className="px-3 py-2 font-medium">
                {heading}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {workloads.map((workload) => (
            <tr key={`${workload.namespace}/${workload.name}`} className="border-b border-border last:border-b-0">
              <td className="px-3 py-2">
                <Link href={`/services/${workload.serviceId}`} className="font-medium hover:underline">
                  {workload.name}
                </Link>
              </td>
              <td className="px-3 py-2 font-mono text-xs">{workload.namespace}</td>
              <td className="px-3 py-2 font-mono text-xs">{workload.version || "—"}</td>
              <td className="px-3 py-2 font-mono tabular-nums">{workload.desired}</td>
              <td className="px-3 py-2 font-mono tabular-nums">{workload.ready}</td>
              <td className="px-3 py-2 font-mono tabular-nums">{workload.restarts}</td>
              <td className="px-3 py-2">
                <StatusBadge value={workload.status} />
              </td>
              <td className="px-3 py-2 text-xs text-muted-foreground">{formatAgo(workload.lastObserved)}</td>
            </tr>
          ))}
        </tbody>
      </table>
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
