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
          {["Service", "Status", "Availability", "P95 latency", "Error rate", "Last deployment"].map((heading, index) => (
            <TableHead
              key={heading}
              className={`sticky top-0 z-10 h-9 bg-secondary text-[11px] font-semibold uppercase tracking-[0.05em] text-muted-foreground ${index >= 2 && index <= 4 ? "text-right" : ""}`}
            >
              {heading}
            </TableHead>
          ))}
        </TableRow>
      </TableHeader>
      <TableBody>
        {services.map((service) => (
          <TableRow key={service.id}>
            <TableCell>
              <Link href={`/services/${service.id}`} className="block max-w-[16rem] truncate font-medium hover:underline" title={service.name}>
                {service.name}
              </Link>
              <div className="text-[11px] text-muted-foreground">{service.namespace}</div>
            </TableCell>
            <TableCell>
              <StatusBadge value={service.status} />
            </TableCell>
            <TableCell className="text-right font-mono text-xs tabular-nums">{formatPercent(service.availability)}</TableCell>
            <TableCell className={service.p95LatencyMs >= 1000 ? "text-right font-mono text-xs tabular-nums text-status-warning" : "text-right font-mono text-xs tabular-nums"}>
              {formatLatency(service.p95LatencyMs)}
            </TableCell>
            <TableCell className={service.errorRate >= 1 ? "text-right font-mono text-xs tabular-nums text-status-danger" : "text-right font-mono text-xs tabular-nums"}>
              {formatPercent(service.errorRate)}
            </TableCell>
            <TableCell>
              <div className="inline-flex h-6 items-center rounded-md bg-secondary px-1.5 font-mono text-[11px]">{service.version}</div>
              <div className="mt-1 text-[11px] text-muted-foreground">{formatAgo(service.lastDeployment, now)}</div>
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
