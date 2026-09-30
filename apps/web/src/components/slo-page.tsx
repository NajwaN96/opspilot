"use client";

import Link from "next/link";
import { StatusBadge } from "@/components/status-badge";
import { ErrorBlock, LoadingBlock, PageHeader } from "@/components/states";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { formatPercent } from "@/lib/format";
import { isPortfolioRuntime } from "@/lib/runtime";
import { useApi } from "@/lib/use-api";
import type { Service } from "@/lib/types";

export function SloPage() {
  const services = useApi<Service[]>("/api/v1/services", 4000);
  if (services.loading && !services.data) return <LoadingBlock />;
  if (!services.data) return <ErrorBlock message={services.error ?? "SLO data unavailable"} onRetry={() => void services.reload()} />;

  const within = services.data.filter((service) => service.slo.withinSLO).length;
  return (
    <div>
      <PageHeader
        kicker="30-day window"
        title="SLOs"
        description={
          isPortfolioRuntime()
            ? `${within} of ${services.data.length} sample services are inside the labeled objective. Error budget and burn rate here are sanitized examples, not a Prometheus query.`
            : `${within} of ${services.data.length} records are inside their objective. payment-api's live error rate and p95 are on its service page. Budget remaining is the fraction of the error budget still unused.`
        }
      />
      <div className="surface-card overflow-hidden rounded-xl">
        <Table>
          <caption className="sr-only">Service level objectives</caption>
          <TableHeader>
            <TableRow>
              {["Service", "Objective", "Compliance", "Error budget left", "Burn rate", "State"].map((heading) => (
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
                  <Link className="font-medium hover:underline" href={`/services/${service.id}`}>
                    {service.name}
                  </Link>
                </TableCell>
                <TableCell className="font-mono">{formatPercent(service.slo.objective)}</TableCell>
                <TableCell className="font-mono">{formatPercent(service.slo.compliance)}</TableCell>
                <TableCell className="font-mono">{formatPercent(service.slo.errorBudgetRemaining, 0)}</TableCell>
                <TableCell className={`font-mono ${service.slo.burnRate >= 2 ? "text-status-danger" : ""}`}>{service.slo.burnRate.toFixed(1)}x</TableCell>
                <TableCell>
                  <StatusBadge value={service.slo.withinSLO ? "healthy" : "critical"} />
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
    </div>
  );
}
