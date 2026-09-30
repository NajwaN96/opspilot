"use client";

import Link from "next/link";
import { StatusBadge } from "@/components/status-badge";
import { EmptyBlock, ErrorBlock, LoadingBlock, PageHeader } from "@/components/states";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { formatAgo, formatDuration } from "@/lib/format";
import { useApi } from "@/lib/use-api";
import type { Incident } from "@/lib/types";

export function IncidentList() {
  const incidents = useApi<Incident[]>("/api/v1/incidents", 4000);
  if (incidents.loading && !incidents.data) return <LoadingBlock />;
  if (!incidents.data) return <ErrorBlock message={incidents.error ?? "Incidents unavailable"} onRetry={() => void incidents.reload()} />;

  return (
    <div>
      <PageHeader kicker="production-01" title="Incidents" description="Open and recently resolved incidents. INC-142 is the active investigation." />
      <div className="surface-card overflow-hidden rounded-xl">
        {incidents.data.length === 0 ? (
          <EmptyBlock title="No incidents" detail="The cluster has nothing recorded in this window." />
        ) : (
          <Table>
            <caption className="sr-only">Incidents</caption>
            <TableHeader>
              <TableRow>
                {["ID", "Severity", "Service", "Title", "Status", "Started", "Duration"].map((heading) => (
                  <TableHead key={heading} className="h-8 text-[11px] uppercase tracking-wide text-muted-foreground">
                    {heading}
                  </TableHead>
                ))}
              </TableRow>
            </TableHeader>
            <TableBody>
              {incidents.data.map((incident) => (
                <TableRow key={incident.id}>
                  <TableCell>
                    <Link href={`/incidents/${incident.id}`} className="font-mono font-medium hover:underline">
                      {incident.id}
                    </Link>
                  </TableCell>
                  <TableCell>
                    <StatusBadge value={incident.severity} />
                  </TableCell>
                  <TableCell>
                    <Link href={`/services/${incident.serviceId}`} className="hover:underline">
                      {incident.serviceName}
                    </Link>
                  </TableCell>
                  <TableCell>
                    <Link href={`/incidents/${incident.id}`} className="hover:underline">
                      {incident.title}
                    </Link>
                  </TableCell>
                  <TableCell>
                    <StatusBadge value={incident.status} />
                  </TableCell>
                  <TableCell className="font-mono text-xs">{formatAgo(incident.startedAt)}</TableCell>
                  <TableCell className="font-mono text-xs">{formatDuration(incident.durationSec * 1000)}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </div>
    </div>
  );
}
