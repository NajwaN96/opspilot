import Link from "next/link";
import { StatusBadge } from "@/components/status-badge";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { formatAgo, formatLatency, formatPercent } from "@/lib/format";
import type { Service } from "@/lib/types";

export function ServiceTable({ services, now }: { services: Service[]; now?: number }) {
  return (
    <Table>
      <caption className="sr-only">Service health</caption>
      <TableHeader>
        <TableRow>
          {["Service", "Status", "Availability", "P95 latency", "Error rate", "Last deployment"].map((heading) => (
            <TableHead key={heading} className="h-9 bg-secondary/80 text-[11px] font-semibold uppercase tracking-[0.05em] text-muted-foreground">
              {heading}
            </TableHead>
          ))}
        </TableRow>
      </TableHeader>
      <TableBody>
        {services.map((service) => (
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
            <TableCell className="font-mono text-xs tabular-nums">{formatPercent(service.availability)}</TableCell>
            <TableCell className={service.p95LatencyMs >= 1000 ? "font-mono text-xs tabular-nums text-status-warning" : "font-mono text-xs tabular-nums"}>
              {formatLatency(service.p95LatencyMs)}
            </TableCell>
            <TableCell className={service.errorRate >= 1 ? "font-mono text-xs tabular-nums text-status-danger" : "font-mono text-xs tabular-nums"}>
              {formatPercent(service.errorRate)}
            </TableCell>
            <TableCell>
              <div className="font-mono text-xs">{service.version}</div>
              <div className="text-[11px] text-muted-foreground">{formatAgo(service.lastDeployment, now)}</div>
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
