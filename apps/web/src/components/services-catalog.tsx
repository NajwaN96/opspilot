"use client";

import Link from "next/link";
import { StatusBadge } from "@/components/status-badge";
import { ErrorBlock, LoadingBlock, PageHeader } from "@/components/states";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { formatLatency, formatPercent } from "@/lib/format";
import { useApi } from "@/lib/use-api";
import type { Service } from "@/lib/types";

export function ServicesCatalog() {
  const services = useApi<Service[]>("/api/v1/services", 4000);
  if (services.loading && !services.data) return <LoadingBlock />;
  if (!services.data) return <ErrorBlock message={services.error ?? "Services unavailable"} onRetry={() => void services.reload()} />;

  return (
    <div>
      <PageHeader kicker="production-01" title="Services" description="Catalog of workloads on the simulated cluster." />
      <div className="border border-border bg-card">
        <Table>
          <caption className="sr-only">Service catalog</caption>
          <TableHeader>
            <TableRow>
              {["Service", "Status", "Owner", "Runtime", "Version", "Availability", "P95 latency", "Error rate", "SLO"].map((heading) => (
                <TableHead key={heading} className="h-8 text-[11px] uppercase tracking-wide text-muted-foreground">
                  {heading}
                </TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {services.data.map((service) => (
              <TableRow key={service.id}>
                <TableCell>
                  <Link href={`/services/${service.id}`} className="font-medium hover:underline">
                    {service.name}
                  </Link>
                  <div className="text-[11px] text-muted-foreground">{service.namespace}</div>
                </TableCell>
                <TableCell>
                  <StatusBadge value={service.status} />
                </TableCell>
                <TableCell>{service.owner}</TableCell>
                <TableCell>{service.runtime}</TableCell>
                <TableCell className="font-mono text-xs">{service.version}</TableCell>
                <TableCell className="font-mono tabular-nums">{formatPercent(service.availability)}</TableCell>
                <TableCell className="font-mono tabular-nums">{formatLatency(service.p95LatencyMs)}</TableCell>
                <TableCell className="font-mono tabular-nums">{formatPercent(service.errorRate)}</TableCell>
                <TableCell>
                  <span className={service.slo.withinSLO ? "text-emerald-300" : "text-red-300"}>
                    {formatPercent(service.slo.compliance)}
                  </span>
                  <div className="text-[11px] text-muted-foreground">obj {formatPercent(service.slo.objective)}</div>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
    </div>
  );
}
