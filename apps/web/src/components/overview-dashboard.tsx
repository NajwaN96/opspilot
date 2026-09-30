"use client";

import type { ReactNode } from "react";
import Link from "next/link";
import { AlertTriangle, ArrowUpRight, Cloud, Compass, Shield } from "lucide-react";
import { ErrorBlock, LoadingBlock, PageHeader, Panel } from "@/components/states";
import { ServiceTable } from "@/components/service-table";
import { StatusBadge } from "@/components/status-badge";
import { formatAgo, formatDuration, formatLatency, formatPercent } from "@/lib/format";
import { useApi } from "@/lib/use-api";
import type { Cluster, Incident, KubernetesStatus, Service, TelemetrySnapshot, TelemetryStatus, Workload } from "@/lib/types";
import { isPortfolioRuntime } from "@/lib/runtime";
import { venueLabel } from "@/lib/venue";
import { cn } from "@/lib/utils";

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
  const sloRatio = health.servicesTotal > 0 ? (health.servicesWithinSLO / health.servicesTotal) * 100 : 0;
  const healthyRatio = health.servicesTotal > 0 ? (health.servicesHealthy / health.servicesTotal) * 100 : 0;
  return (
    <div>
      <PageHeader
        kicker={isPortfolioRuntime() ? "Portfolio sample" : cluster.name}
        title="Overview"
        description={
          isPortfolioRuntime()
            ? "Operational health for the public portfolio. These figures are sanitized sample data, not live Kubernetes, Prometheus, or Jaeger telemetry."
            : "Operational health and reliability signals for the local lab. Simulated production-01 stays separate from demo-shop."
        }
      />
      {active.length > 0 ? (
        <div className="mb-5 space-y-2">
          {active.map((incident) => (
            <Link
              key={incident.id}
              href={`/incidents/${incident.id}`}
              className="flex items-start gap-3 rounded-xl border border-status-danger/15 bg-status-danger-soft px-4 py-3 transition-colors duration-150 hover:bg-[#ffe8e8]"
            >
              <AlertTriangle className="mt-0.5 size-4 shrink-0 text-status-danger" strokeWidth={1.75} aria-hidden />
              <span className="min-w-0 flex-1">
                <span className="flex flex-wrap items-center gap-2">
                  <StatusBadge value={incident.severity} />
                  <span className="text-sm font-medium text-foreground">{incident.title}</span>
                </span>
                <span className="mt-1 flex flex-wrap items-center justify-between gap-x-3 gap-y-1">
                  <span className="text-xs text-muted-foreground">
                    {incident.serviceName} · {formatDuration(incident.durationSec * 1000)} ·{" "}
                    {incident.origin === "detection-engine" ? "Detection Engine" : "Simulation"}
                  </span>
                  <span className="inline-flex items-center gap-1 text-sm font-medium text-status-info">
                    View incident
                    <ArrowUpRight className="size-3.5" aria-hidden />
                  </span>
                </span>
              </span>
            </Link>
          ))}
        </div>
      ) : null}
      <div className="mb-5 grid gap-3 sm:grid-cols-2 xl:grid-cols-5">
        <MetricTile label="Cluster health" value={health.state} hint={cluster.kubernetesVersion} tone={health.state} />
        <MetricTile
          label="Active incidents"
          value={String(health.activeIncidents)}
          hint={health.activeIncidents === 1 ? "SEV-2 open" : "Open right now"}
          tone={health.activeIncidents > 0 ? "critical" : "healthy"}
        />
        <MetricTile
          label="SLO compliance"
          value={`${health.servicesWithinSLO} / ${health.servicesTotal}`}
          hint={`${formatPercent(health.sloCompliance)} average`}
          progress={sloRatio}
          progressTone={sloRatio >= 100 ? "good" : "warn"}
        />
        <MetricTile
          label="Services"
          value={`${health.servicesHealthy} / ${health.servicesTotal}`}
          hint="Healthy services"
          progress={healthyRatio}
          progressTone={healthyRatio >= 100 ? "good" : "warn"}
        />
        <MetricTile label="Deployments today" value={String(health.deploymentsToday)} hint="Last 24 hours" />
      </div>
      <div className="grid gap-4 xl:grid-cols-[minmax(0,1.7fr)_minmax(300px,0.85fr)]">
        <Panel
          prominence="primary"
          title="Service health"
          action={<span className="text-[12px] text-muted-foreground">Source: Simulation</span>}
          padded={false}
        >
          <ServiceTable services={simulatedServices} />
        </Panel>
        <Panel
          prominence="primary"
          title="Active incidents"
          action={
            <Link href="/incidents" className="text-[12px] font-medium text-status-info hover:underline">
              View all
            </Link>
          }
          padded={false}
        >
          {active.length === 0 ? (
            <p className="px-4 py-6 text-sm text-muted-foreground">No active incidents. {cluster.name} is clear.</p>
          ) : (
            <ul>
              {active.map((incident) => (
                <li key={incident.id} className="border-b border-border last:border-b-0">
                  <Link href={`/incidents/${incident.id}`} className="block px-4 py-3 transition-colors duration-150 hover:bg-[#f8fafc]">
                    <div className="flex items-center gap-2">
                      <StatusBadge value={incident.severity} />
                      <StatusBadge value={incident.status} />
                      <span className="ml-auto font-mono text-[11px] text-muted-foreground">{incident.id}</span>
                    </div>
                    <span className="mt-2 block text-sm font-medium text-foreground">{incident.title}</span>
                    <span className="mt-1 block text-xs text-muted-foreground">
                      {incident.serviceName} · {formatDuration(incident.durationSec * 1000)}
                    </span>
                  </Link>
                </li>
              ))}
            </ul>
          )}
        </Panel>
      </div>
      <div className="mt-4">
        <Panel
          title="Kubernetes workloads"
          action={<span className="text-[12px] text-muted-foreground">Source: Kubernetes · {kubernetes.data?.namespace ?? "demo-shop"}</span>}
          padded={false}
        >
          <WorkloadTable workloads={workloads.data} connectivity={kubernetes.data?.connectivity} message={workloads.error ?? kubernetes.data?.message} />
        </Panel>
      </div>
      <div className="mt-4">
        <EnvironmentStrip status={kubernetes.data} telemetry={telemetry.data} payment={payment.data} />
      </div>
      <div className="mt-4 grid gap-3 md:grid-cols-3">
        <InfoNote
          icon={Cloud}
          title={isPortfolioRuntime() ? "Public portfolio" : "Local lab"}
          body={
            isPortfolioRuntime()
              ? "This page is a sanitized sample on Lambda. It is not connected to k3d, Prometheus, or the executor."
              : "k3d opspilot-dev is live. payment-api metrics below are Prometheus. INC-142 remains a simulated incident."
          }
        />
        <InfoNote icon={Shield} title="Plan state" body="EKS, its node group, and ECR exist as OpenTofu. They are not applied. The paid override stays unset." />
        <InfoNote
          icon={Compass}
          title="Demo tour"
          body={
            <>
              Start at the <Link className="text-status-info hover:underline" href="/developer-portal">developer portal</Link>, then{" "}
              <Link className="text-status-info hover:underline" href="/slos">SLOs</Link>,{" "}
              <Link className="text-status-info hover:underline" href="/incidents">incidents</Link>,{" "}
              <Link className="text-status-info hover:underline" href="/rollouts">rollouts</Link>, and{" "}
              <Link className="text-status-info hover:underline" href="/security">security</Link>.
            </>
          }
        />
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
    <section className="surface-card rounded-xl px-4 py-3">
      <div className="mb-3 text-[11px] font-semibold uppercase tracking-[0.06em] text-muted-foreground">Integration health</div>
      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <Integration label="Prometheus" value={telemetry?.prometheus} />
        <Integration label="OpenTelemetry" value={telemetry?.opentelemetry} />
        <Integration label="Trace backend" value={telemetry?.traces} />
        <Integration label="payment-api" detail={paymentLine(payment)} value={payment?.available ? "connected" : "unavailable"} />
      </div>
      <p className="mt-3 text-xs text-muted-foreground">
        {venueLabel(status)} · {status?.cluster ?? "opspilot-dev"} · {status?.namespace ?? "demo-shop"}
      </p>
    </section>
  );
}

function paymentLine(payment?: TelemetrySnapshot | null): string {
  if (!payment?.available) return "Telemetry unavailable";
  return `${payment.requestRate.toFixed(2)} req/s · ${formatPercent(payment.errorRate * 100)} errors · p95 ${formatLatency(payment.p95LatencyMs)}`;
}

function Integration({ label, value, detail }: { label: string; value?: string; detail?: string }) {
  const state = value === "connected" ? "connected" : value === "unavailable" || !value ? "unavailable" : value;
  return (
    <div>
      <div className="text-[12px] font-medium text-foreground">{label}</div>
      <div className="mt-1">
        <StatusBadge value={state} />
      </div>
      {detail ? <p className="mt-1 font-mono text-[12px] text-muted-foreground">{detail}</p> : null}
    </div>
  );
}

function WorkloadTable({ workloads, connectivity, message }: { workloads?: Workload[] | null; connectivity?: string; message?: string }) {
  if (connectivity === "disconnected") {
    return <p className="px-4 py-5 text-sm text-muted-foreground">{message || "Local cluster is disconnected. No workloads are invented for this view."}</p>;
  }
  if (!workloads) {
    return <p className="px-4 py-5 text-sm text-muted-foreground">Waiting for the Kubernetes read.</p>;
  }
  if (workloads.length === 0) {
    return <p className="px-4 py-5 text-sm text-muted-foreground">No Deployments in the discovery namespace.</p>;
  }
  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-border bg-secondary/70 text-left text-[11px] font-semibold uppercase tracking-[0.05em] text-muted-foreground">
            {["Workload", "Namespace", "Version", "Desired", "Ready", "Restarts", "Status", "Last observed"].map((heading) => (
              <th key={heading} className="px-4 py-2.5 font-semibold">
                {heading}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {workloads.map((workload) => (
            <tr key={`${workload.namespace}/${workload.name}`} className="border-b border-border/80 transition-colors duration-150 last:border-b-0 hover:bg-[#f8fafc]">
              <td className="px-4 py-2.5">
                <Link href={`/services/${workload.serviceId}`} className="font-medium hover:underline">
                  {workload.name}
                </Link>
              </td>
              <td className="px-4 py-2.5 font-mono text-xs">{workload.namespace}</td>
              <td className="px-4 py-2.5 font-mono text-xs">{workload.version || "—"}</td>
              <td className="px-4 py-2.5 font-mono text-xs tabular-nums">{workload.desired}</td>
              <td className="px-4 py-2.5 font-mono text-xs tabular-nums">{workload.ready}</td>
              <td className="px-4 py-2.5 font-mono text-xs tabular-nums">{workload.restarts}</td>
              <td className="px-4 py-2.5">
                <StatusBadge value={workload.status} />
              </td>
              <td className="px-4 py-2.5 text-xs text-muted-foreground">{formatAgo(workload.lastObserved)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function MetricTile({
  label,
  value,
  hint,
  tone,
  progress,
  progressTone = "good",
}: {
  label: string;
  value: string;
  hint: string;
  tone?: string;
  progress?: number;
  progressTone?: "good" | "warn";
}) {
  const valueTone =
    tone === "critical"
      ? "text-status-danger"
      : tone === "degraded"
        ? "text-status-warning"
        : tone === "healthy"
          ? "text-status-success"
          : "text-foreground";
  return (
    <div className="surface-card rounded-xl px-4 py-3">
      <div className="text-[11px] font-semibold uppercase tracking-[0.06em] text-muted-foreground">{label}</div>
      <div className={cn("mt-2 text-[28px] font-semibold capitalize tracking-tight tabular-nums", valueTone)}>{value}</div>
      {progress != null ? (
        <div className="mt-2 h-1 overflow-hidden rounded-full bg-muted" aria-hidden>
          <div
            className={cn("h-full rounded-full", progressTone === "warn" ? "bg-[var(--status-dot-warning)]" : "bg-[var(--status-dot-success)]")}
            style={{ width: `${Math.max(0, Math.min(100, progress))}%` }}
          />
        </div>
      ) : null}
      <div className="mt-1.5 text-xs text-muted-foreground">{hint}</div>
    </div>
  );
}

function InfoNote({ icon: Icon, title, body }: { icon: typeof Cloud; title: string; body: ReactNode }) {
  return (
    <div className="rounded-xl border border-border bg-white/70 px-4 py-3">
      <div className="flex items-center gap-2">
        <Icon className="size-3.5 text-muted-foreground" strokeWidth={1.75} aria-hidden />
        <h2 className="text-[13px] font-semibold">{title}</h2>
      </div>
      <p className="mt-1.5 text-[13px] leading-relaxed text-muted-foreground">{body}</p>
    </div>
  );
}
