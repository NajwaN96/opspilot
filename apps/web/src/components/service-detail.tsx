"use client";

import Link from "next/link";
import { MetricChart } from "@/components/charts";
import { StatusBadge } from "@/components/status-badge";
import { EmptyBlock, ErrorBlock, LoadingBlock, PageHeader, Panel } from "@/components/states";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { formatAgo, formatLatency, formatPercent } from "@/lib/format";
import { useApi } from "@/lib/use-api";
import type { Incident, Service } from "@/lib/types";

export function ServiceDetail({ id }: { id: string }) {
  const service = useApi<Service>(`/api/v1/services/${id}`, 4000);
  const incidents = useApi<Incident[]>("/api/v1/incidents", 4000);

  if (service.loading && !service.data) return <LoadingBlock label="Loading service" />;
  if (!service.data) return <ErrorBlock message={service.error ?? "Service not found"} onRetry={() => void service.reload()} />;

  const svc = service.data;
  const related = incidents.data?.filter((incident) => incident.serviceId === svc.id) ?? [];

  return (
    <div>
      <PageHeader
        kicker={`${svc.namespace} / ${svc.clusterId}`}
        title={svc.name}
        description={svc.description}
        actions={<StatusBadge value={svc.status} />}
      />
      <dl className="mb-3 grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <Fact label="Owner" value={svc.owner} />
        <Fact label="Runtime" value={svc.runtime} />
        <Fact label="Version" value={`${svc.version} ← ${svc.previousVersion}`} mono />
        <Fact label="Replicas" value={`${svc.replicas.ready}/${svc.replicas.desired}`} mono />
      </dl>
      <Tabs defaultValue="overview">
        <TabsList variant="line">
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="reliability">Reliability</TabsTrigger>
          <TabsTrigger value="deployments">Deployments</TabsTrigger>
          <TabsTrigger value="incidents">Incidents</TabsTrigger>
          <TabsTrigger value="dependencies">Dependencies</TabsTrigger>
        </TabsList>
        <TabsContent value="overview" className="mt-3">
          <div className="grid gap-3 md:grid-cols-4">
            <Fact label="Availability" value={formatPercent(svc.availability)} mono />
            <Fact label="P95 latency" value={formatLatency(svc.p95LatencyMs)} mono />
            <Fact label="Error rate" value={formatPercent(svc.errorRate)} mono />
            <Fact label="Last deployment" value={formatAgo(svc.lastDeployment)} />
          </div>
          <div className="mt-3">
            <Panel title="Open incidents" padded={false}>
              {svc.openIncidents && svc.openIncidents.length > 0 ? (
                <ul>
                  {svc.openIncidents.map((incidentId) => (
                    <li key={incidentId} className="px-3 py-2">
                      <Link className="font-mono text-sm hover:underline" href={`/incidents/${incidentId}`}>
                        {incidentId}
                      </Link>
                    </li>
                  ))}
                </ul>
              ) : (
                <EmptyBlock title="No open incidents" detail="This service is not attached to an active incident." />
              )}
            </Panel>
          </div>
        </TabsContent>
        <TabsContent value="reliability" className="mt-3">
          <Panel title={`SLO · ${svc.slo.window} objective ${formatPercent(svc.slo.objective)}`}>
            <div className="mb-3 grid gap-3 sm:grid-cols-4">
              <Fact label="Compliance" value={formatPercent(svc.slo.compliance)} mono />
              <Fact label="Error budget left" value={formatPercent(svc.slo.errorBudgetRemaining, 0)} mono />
              <Fact label="Burn rate" value={`${svc.slo.burnRate.toFixed(1)}x`} mono />
              <Fact label="Within SLO" value={svc.slo.withinSLO ? "Yes" : "No"} />
            </div>
            <MetricChart data={svc.metrics ?? []} />
            <p className="mt-2 text-[11px] text-muted-foreground">Simulated samples. p95 uses the left axis, error rate the right.</p>
          </Panel>
        </TabsContent>
        <TabsContent value="deployments" className="mt-3">
          <DeploymentTable deployments={svc.deployments ?? []} />
        </TabsContent>
        <TabsContent value="incidents" className="mt-3">
          <Panel title="Incident history" padded={false}>
            {incidents.loading && !incidents.data ? (
              <LoadingBlock label="Loading incidents" />
            ) : related.length === 0 ? (
              <EmptyBlock title="No incidents" detail="This service has no recorded incidents in the simulation." />
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    {["ID", "Severity", "Title", "Status", "Started"].map((heading) => (
                      <TableHead key={heading} className="h-8 text-[11px] uppercase tracking-wide text-muted-foreground">
                        {heading}
                      </TableHead>
                    ))}
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {related.map((incident) => (
                    <TableRow key={incident.id}>
                      <TableCell>
                        <Link className="font-mono hover:underline" href={`/incidents/${incident.id}`}>
                          {incident.id}
                        </Link>
                      </TableCell>
                      <TableCell>
                        <StatusBadge value={incident.severity} />
                      </TableCell>
                      <TableCell>{incident.title}</TableCell>
                      <TableCell>
                        <StatusBadge value={incident.status} />
                      </TableCell>
                      <TableCell className="font-mono text-xs">{formatAgo(incident.startedAt)}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </Panel>
        </TabsContent>
        <TabsContent value="dependencies" className="mt-3">
          <Panel title="Dependencies" padded={false}>
            <Table>
              <TableHeader>
                <TableRow>
                  {["Name", "Kind", "Relation", "Status"].map((heading) => (
                    <TableHead key={heading} className="h-8 text-[11px] uppercase tracking-wide text-muted-foreground">
                      {heading}
                    </TableHead>
                  ))}
                </TableRow>
              </TableHeader>
              <TableBody>
                {(svc.dependencies ?? []).map((dependency) => (
                  <TableRow key={`${dependency.relation}-${dependency.id}`}>
                    <TableCell className="font-medium">
                      {dependency.kind === "service" ? (
                        <Link className="hover:underline" href={`/services/${dependency.id}`}>
                          {dependency.name}
                        </Link>
                      ) : (
                        dependency.name
                      )}
                    </TableCell>
                    <TableCell className="capitalize">{dependency.kind}</TableCell>
                    <TableCell className="capitalize">{dependency.relation}</TableCell>
                    <TableCell>
                      <StatusBadge value={dependency.status} />
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </Panel>
        </TabsContent>
      </Tabs>
    </div>
  );
}

function Fact({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div className="border border-border bg-card px-3 py-2">
      <div className="text-[11px] uppercase tracking-wide text-muted-foreground">{label}</div>
      <div className={mono ? "mt-1 font-mono text-sm tabular-nums" : "mt-1 text-sm"}>{value}</div>
    </div>
  );
}

function DeploymentTable({ deployments }: { deployments: Service["deployments"] }) {
  if (!deployments || deployments.length === 0) {
    return (
      <Panel title="Deployments" padded={false}>
        <EmptyBlock title="No deployments" detail="This service has no recorded rollouts." />
      </Panel>
    );
  }
  return (
    <Panel title="Deployments" padded={false}>
      <Table>
        <TableHeader>
          <TableRow>
            {["When", "Version", "Previous", "Status", "Actor", "Change"].map((heading) => (
              <TableHead key={heading} className="h-8 text-[11px] uppercase tracking-wide text-muted-foreground">
                {heading}
              </TableHead>
            ))}
          </TableRow>
        </TableHeader>
        <TableBody>
          {deployments.map((deployment) => (
            <TableRow key={deployment.id}>
              <TableCell className="font-mono text-xs">
                {deployment.clock}
                <div className="text-muted-foreground">{formatAgo(deployment.at)}</div>
              </TableCell>
              <TableCell className="font-mono">{deployment.version}</TableCell>
              <TableCell className="font-mono text-muted-foreground">{deployment.previousVersion}</TableCell>
              <TableCell>
                <StatusBadge value={deployment.status} />
              </TableCell>
              <TableCell>{deployment.actor}</TableCell>
              <TableCell className="max-w-sm whitespace-normal text-muted-foreground">{deployment.change}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </Panel>
  );
}
