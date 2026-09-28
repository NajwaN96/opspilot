"use client";

import { StatusBadge } from "@/components/status-badge";
import { ErrorBlock, LoadingBlock, PageHeader, Panel } from "@/components/states";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { useApi } from "@/lib/use-api";
import type { Cluster } from "@/lib/types";

export function InfrastructurePage() {
  const clusters = useApi<Cluster[]>("/api/v1/clusters", 5000);
  if (clusters.loading && !clusters.data) return <LoadingBlock />;
  const cluster = clusters.data?.[0];
  if (!cluster) return <ErrorBlock message={clusters.error ?? "Cluster inventory unavailable"} onRetry={() => void clusters.reload()} />;

  return (
    <div>
      <PageHeader
        kicker={cluster.provider}
        title="Infrastructure"
        description={`${cluster.name} is a simulated ${cluster.kubernetesVersion} cluster in ${cluster.region}. Node metrics are not collected from kubelet.`}
      />
      <div className="grid gap-3 lg:grid-cols-2">
        <Panel title="Control plane" padded={false}>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="h-8 text-[11px] uppercase tracking-wide text-muted-foreground">Component</TableHead>
                <TableHead className="h-8 text-[11px] uppercase tracking-wide text-muted-foreground">Status</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {cluster.controlPlane.map((component) => (
                <TableRow key={component.name}>
                  <TableCell className="font-mono text-xs">{component.name}</TableCell>
                  <TableCell>
                    <StatusBadge value={component.status} />
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </Panel>
        <Panel title="Namespaces" padded={false}>
          <Table>
            <TableHeader>
              <TableRow>
                {["Namespace", "Services", "Status"].map((heading) => (
                  <TableHead key={heading} className="h-8 text-[11px] uppercase tracking-wide text-muted-foreground">
                    {heading}
                  </TableHead>
                ))}
              </TableRow>
            </TableHeader>
            <TableBody>
              {cluster.namespaces.map((namespace) => (
                <TableRow key={namespace.name}>
                  <TableCell className="font-mono">{namespace.name}</TableCell>
                  <TableCell className="font-mono">{namespace.services}</TableCell>
                  <TableCell>
                    <StatusBadge value={namespace.status} />
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </Panel>
      </div>
      <div className="mt-3">
        <Panel title="Nodes" padded={false}>
          <Table>
            <TableHeader>
              <TableRow>
                {["Node", "Status", "Role", "Zone", "CPU", "Memory", "Pods", "Kubelet"].map((heading) => (
                  <TableHead key={heading} className="h-8 text-[11px] uppercase tracking-wide text-muted-foreground">
                    {heading}
                  </TableHead>
                ))}
              </TableRow>
            </TableHeader>
            <TableBody>
              {cluster.nodes.map((node) => (
                <TableRow key={node.name}>
                  <TableCell className="font-mono">{node.name}</TableCell>
                  <TableCell>
                    <StatusBadge value={node.status === "Ready" ? "ready" : node.status} />
                  </TableCell>
                  <TableCell>{node.role}</TableCell>
                  <TableCell>{node.zone}</TableCell>
                  <TableCell className="font-mono">{node.cpuPercent}%</TableCell>
                  <TableCell className="font-mono">{node.memoryPercent}%</TableCell>
                  <TableCell className="font-mono">
                    {node.pods}/{node.podCapacity}
                  </TableCell>
                  <TableCell className="font-mono text-xs">{node.kubeletVersion}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </Panel>
      </div>
    </div>
  );
}
