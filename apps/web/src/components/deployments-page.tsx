"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { StatusBadge } from "@/components/status-badge";
import { ErrorBlock, LoadingBlock, PageHeader } from "@/components/states";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { apiGet } from "@/lib/api";
import { formatAgo } from "@/lib/format";
import type { Deployment, Service } from "@/lib/types";

export function DeploymentsPage() {
  const [rows, setRows] = useState<Deployment[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let active = true;
    const load = async () => {
      try {
        const list = await apiGet<Service[]>("/api/v1/services");
        const detailed = await Promise.all(list.map((service) => apiGet<Service>(`/api/v1/services/${service.id}`)));
        const deployments = detailed
          .flatMap((service) => service.deployments ?? [])
          .sort((a, b) => Date.parse(b.at) - Date.parse(a.at));
        if (active) {
          setRows(deployments);
          setError(null);
        }
      } catch (err) {
        if (active) setError(err instanceof Error ? err.message : "Deployments unavailable");
      }
    };
    void load();
    const id = setInterval(() => void load(), 4000);
    return () => {
      active = false;
      clearInterval(id);
    };
  }, []);

  if (!rows && !error) return <LoadingBlock label="Loading deployments" />;
  if (!rows) return <ErrorBlock message={error ?? "Deployments unavailable"} />;

  return (
    <div>
      <PageHeader
        kicker="production-01"
        title="Deployments"
        description="Rollouts recorded by the simulated cluster. The payment-api v1.8.2 rollout is the suspect change for INC-142."
      />
      <div className="surface-card overflow-hidden rounded-xl">
        <Table>
          <caption className="sr-only">Deployments</caption>
          <TableHeader>
            <TableRow>
              {["When", "Service", "Version", "Previous", "Status", "Strategy", "Actor", "Change"].map((heading) => (
                <TableHead key={heading} className="h-8 text-[11px] uppercase tracking-wide text-muted-foreground">
                  {heading}
                </TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {rows.map((deployment) => (
              <TableRow key={deployment.id}>
                <TableCell className="font-mono text-xs">
                  {deployment.clock}
                  <div className="text-muted-foreground">{formatAgo(deployment.at)}</div>
                </TableCell>
                <TableCell>
                  <Link className="hover:underline" href={`/services/${deployment.serviceId}`}>
                    {deployment.serviceName}
                  </Link>
                </TableCell>
                <TableCell className="font-mono">{deployment.version}</TableCell>
                <TableCell className="font-mono text-muted-foreground">{deployment.previousVersion}</TableCell>
                <TableCell>
                  <StatusBadge value={deployment.status} />
                </TableCell>
                <TableCell>{deployment.strategy}</TableCell>
                <TableCell>{deployment.actor}</TableCell>
                <TableCell className="max-w-xs whitespace-normal text-muted-foreground">{deployment.change}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
    </div>
  );
}
