"use client";

import Link from "next/link";
import { StatusBadge } from "@/components/status-badge";
import { ErrorBlock, LoadingBlock, PageHeader, Panel } from "@/components/states";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { formatAgo } from "@/lib/format";
import { useApi } from "@/lib/use-api";
import type { CloudPortfolio, ClusterEvent, KubernetesStatus, Namespace, ReadyStatus, TelemetryStatus, Workload, WorkloadPod } from "@/lib/types";
import { isPortfolioRuntime } from "@/lib/runtime";
import { clusterModeLabel, venueLabel } from "@/lib/venue";

export function InfrastructurePage() {
  const status = useApi<KubernetesStatus>("/api/v1/kubernetes/status", 5000);
  const namespaces = useApi<Namespace[]>("/api/v1/kubernetes/namespaces", 5000);
  const workloads = useApi<Workload[]>("/api/v1/kubernetes/workloads", 5000);
  const pods = useApi<WorkloadPod[]>("/api/v1/kubernetes/pods", 5000);
  const events = useApi<ClusterEvent[]>("/api/v1/kubernetes/events", 5000);
  const telemetry = useApi<TelemetryStatus>("/api/v1/telemetry/status", 5000);
  const ready = useApi<ReadyStatus>("/ready", 5000);
  const cloud = useApi<CloudPortfolio>("/api/v1/cloud/status", 15000);

  if (status.loading && !status.data) return <LoadingBlock label="Reading cluster" />;
  if (!status.data) return <ErrorBlock message={status.error ?? "Cluster status unavailable"} onRetry={() => void status.reload()} />;

  const live = status.data;
  const portfolio = isPortfolioRuntime() || live.mode === "aws-portfolio-demo";
  const disconnected = live.connectivity === "disconnected";

  return (
    <div>
      <PageHeader
        kicker="Source: Kubernetes"
        title="Infrastructure"
        description={
          portfolio
            ? "This page is the public AWS portfolio. It is not attached to k3d or to EKS. Sample rows below are sanitized. The local reliability lab and the plan-only EKS design stay separate."
            : "Live Kubernetes is local k3d. The Amazon EKS design stays plan-only and is not deployed. The constrained executor cannot target opspilot-aws-dev."
        }
      />
      <Panel title="Cloud" className="mb-3">
        <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
          <Fact
            label="AWS account"
            value={
              cloud.data?.accountState === "connected"
                ? "Connected"
                : cloud.data?.accountState === "portfolio-demo"
                  ? "Portfolio mask only"
                  : "Not connected"
            }
          />
          <Fact label="Account mask" value={cloud.data?.accountMasked || "—"} />
          <Fact label="AWS region" value={cloud.data?.region || "—"} />
          <Fact label="Cloud deployment" value={cloud.data?.cloudDeployment || "PLAN ONLY"} />
          <Fact label="Kubernetes runtime" value={portfolio ? cloud.data?.kubernetesRuntime || "Not connected" : "LOCAL k3d"} />
          <Fact label="Paid cloud resources" value={cloud.data?.paidCloudResources || "BLOCKED"} />
          <Fact label="Intentional AWS infrastructure spend" value={cloud.data?.intentionalInfrastructureSpend || "$0"} />
          <Fact label="EKS" value="plan-only" />
        </div>
        <p className="mt-3 text-xs text-muted-foreground">
          Intentional spend is the infrastructure OpsPilot is allowed to create. It is not a bill from AWS. Alerts do not stop resources by themselves, and credits are not permission to deploy EKS.
        </p>
      </Panel>
      <div className="mb-3 grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <Fact label="Environment" value={venueLabel(live)} />
        <Fact label="Cluster" value={live.cluster} />
        <Fact label="Mode" value={clusterModeLabel(live.mode)} />
        <div className="surface-card overflow-hidden rounded-xl px-3 py-2">
          <div className="text-[11px] uppercase tracking-wide text-muted-foreground">Connectivity</div>
          <div className="mt-1">
            <StatusBadge value={live.connectivity} />
          </div>
        </div>
        <Fact label="Nodes ready" value={`${live.nodesReady}/${live.nodeCount}`} />
      </div>
      <div className="mb-3 grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <ComponentHealth name="PostgreSQL" value={ready.data?.database === "ok" ? "connected" : ready.data?.database} />
        <ComponentHealth name="Prometheus" value={telemetry.data?.prometheus} />
        <ComponentHealth name="OpenTelemetry" value={telemetry.data?.opentelemetry} />
        <ComponentHealth name="Trace backend" value={telemetry.data?.traces} />
      </div>
      {disconnected ? (
        <div className="surface-card overflow-hidden rounded-xl px-3 py-4 text-sm text-muted-foreground">
          {live.message || "The local cluster is disconnected."} Namespace, workload, pod, and event tables stay empty until a real API response arrives.
        </div>
      ) : (
        <div className="space-y-3">
          <Panel title="Namespaces" padded={false}>
            <SimpleTable
              headings={["Namespace", "Deployments", "Status"]}
              rows={(namespaces.data ?? []).map((namespace) => [
                namespace.name,
                String(namespace.services),
                namespace.status,
              ])}
              empty="No namespaces in the discovery scope."
            />
          </Panel>
          <Panel title="Workloads" action={<span className="text-[11px] text-muted-foreground">{live.namespace}</span>} padded={false}>
            <Table>
              <TableHeader>
                <TableRow>
                  {["Workload", "Namespace", "Version", "Desired", "Ready", "Restarts", "Status", "Last observed"].map((heading) => (
                    <TableHead key={heading} className="h-8 text-[11px] uppercase tracking-wide text-muted-foreground">
                      {heading}
                    </TableHead>
                  ))}
                </TableRow>
              </TableHeader>
              <TableBody>
                {(workloads.data ?? []).map((workload) => (
                  <TableRow key={`${workload.namespace}/${workload.name}`}>
                    <TableCell>
                      <Link href={`/services/${workload.serviceId}`} className="font-medium hover:underline">
                        {workload.name}
                      </Link>
                    </TableCell>
                    <TableCell className="font-mono text-xs">{workload.namespace}</TableCell>
                    <TableCell className="font-mono text-xs">{workload.version || "—"}</TableCell>
                    <TableCell className="font-mono">{workload.desired}</TableCell>
                    <TableCell className="font-mono">{workload.ready}</TableCell>
                    <TableCell className="font-mono">{workload.restarts}</TableCell>
                    <TableCell>
                      <StatusBadge value={workload.status} />
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">{formatAgo(workload.lastObserved)}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </Panel>
          <Panel title="Pods" padded={false}>
            <Table>
              <TableHeader>
                <TableRow>
                  {["Pod", "Service", "Status", "Ready", "Restarts", "Node", "Age"].map((heading) => (
                    <TableHead key={heading} className="h-8 text-[11px] uppercase tracking-wide text-muted-foreground">
                      {heading}
                    </TableHead>
                  ))}
                </TableRow>
              </TableHeader>
              <TableBody>
                {(pods.data ?? []).map((pod) => (
                  <TableRow key={`${pod.namespace}/${pod.name}`}>
                    <TableCell className="font-mono text-xs">{pod.name}</TableCell>
                    <TableCell>{pod.service || "—"}</TableCell>
                    <TableCell>
                      <StatusBadge value={pod.status.toLowerCase() === "running" ? "healthy" : pod.status.toLowerCase()} />
                    </TableCell>
                    <TableCell className="font-mono">{pod.ready}</TableCell>
                    <TableCell className="font-mono">{pod.restarts}</TableCell>
                    <TableCell className="font-mono text-xs">{pod.node || "—"}</TableCell>
                    <TableCell className="text-xs text-muted-foreground">{pod.startedAt ? formatAgo(pod.startedAt) : "—"}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </Panel>
          <Panel title="Recent Kubernetes events" padded={false}>
            <Table>
              <TableHeader>
                <TableRow>
                  {["When", "Type", "Reason", "Object", "Message"].map((heading) => (
                    <TableHead key={heading} className="h-8 text-[11px] uppercase tracking-wide text-muted-foreground">
                      {heading}
                    </TableHead>
                  ))}
                </TableRow>
              </TableHeader>
              <TableBody>
                {(events.data ?? []).length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={5} className="text-sm text-muted-foreground">
                      No events in the discovery namespace.
                    </TableCell>
                  </TableRow>
                ) : (
                  (events.data ?? []).map((event) => (
                    <TableRow key={event.id}>
                      <TableCell className="text-xs text-muted-foreground">{formatAgo(event.at)}</TableCell>
                      <TableCell>
                        <StatusBadge value={event.type.toLowerCase() === "warning" ? "warning" : "normal"} />
                      </TableCell>
                      <TableCell className="font-mono text-xs">{event.reason}</TableCell>
                      <TableCell className="font-mono text-xs">{event.object}</TableCell>
                      <TableCell className="max-w-md truncate text-xs">{event.message}</TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </Panel>
        </div>
      )}
      {events.error ? <p className="mt-2 text-xs text-status-danger">{events.error}</p> : null}
    </div>
  );
}

function ComponentHealth({ name, value }: { name: string; value?: string }) {
  const connected = value === "connected" || value === "ok";
  return (
    <div className="surface-card overflow-hidden rounded-xl px-3 py-2">
      <div className="text-[11px] uppercase tracking-wide text-muted-foreground">{name}</div>
      <div className={`mt-1 text-sm ${connected ? "text-status-success" : "text-status-warning"}`}>{connected ? "Connected" : "Unavailable"}</div>
    </div>
  );
}

function Fact({ label, value }: { label: string; value: string }) {
  return (
    <div className="surface-card overflow-hidden rounded-xl px-3 py-2">
      <div className="text-[11px] uppercase tracking-wide text-muted-foreground">{label}</div>
      <div className="mt-1 font-mono text-sm">{value}</div>
    </div>
  );
}

function SimpleTable({ headings, rows, empty }: { headings: string[]; rows: string[][]; empty: string }) {
  if (rows.length === 0) return <p className="px-3 py-4 text-sm text-muted-foreground">{empty}</p>;
  return (
    <Table>
      <TableHeader>
        <TableRow>
          {headings.map((heading) => (
            <TableHead key={heading} className="h-8 text-[11px] uppercase tracking-wide text-muted-foreground">
              {heading}
            </TableHead>
          ))}
        </TableRow>
      </TableHeader>
      <TableBody>
        {rows.map((row) => (
          <TableRow key={row.join("/")}>
            {row.map((cell, index) => (
              <TableCell key={`${row[0]}-${headings[index]}`} className={index === 0 ? "font-mono" : ""}>
                {index === headings.length - 1 ? <StatusBadge value={cell} /> : cell}
              </TableCell>
            ))}
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
