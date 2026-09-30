"use client";

import { MetricChart } from "@/components/charts";
import { StatusBadge } from "@/components/status-badge";
import { EmptyBlock } from "@/components/states";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { formatAgo, formatLatency } from "@/lib/format";
import type { Evidence } from "@/lib/types";

export function EvidencePanel({ evidence }: { evidence: Evidence }) {
  return (
    <Tabs defaultValue="metrics">
      <TabsList variant="line" className="px-3 pt-2">
        <TabsTrigger value="metrics">Metrics</TabsTrigger>
        <TabsTrigger value="logs">Logs</TabsTrigger>
        <TabsTrigger value="traces">Traces</TabsTrigger>
        <TabsTrigger value="events">Kubernetes events</TabsTrigger>
        <TabsTrigger value="deployments">Deployments</TabsTrigger>
      </TabsList>
      <TabsContent value="metrics" className="p-3">
        <MetricChart data={evidence.metrics} />
      </TabsContent>
      <TabsContent value="logs" className="px-3 py-2">
        <ul className="font-mono text-xs">
          {evidence.logs.map((line, index) => {
            const hot = line.message.includes("database connection pool exhausted");
            return (
              <li key={`${line.clock}-${index}`} className={`grid grid-cols-[3.25rem_3.5rem_8rem_minmax(0,1fr)] gap-2 border-b border-border/70 py-1.5 ${hot ? "bg-status-danger-soft" : ""}`}>
                <span className="text-muted-foreground">{line.clock}</span>
                <span className={line.level === "ERROR" ? "text-status-danger" : line.level === "WARN" ? "text-status-warning" : "text-status-info"}>
                  {line.level}
                </span>
                <span className="truncate text-muted-foreground">{line.service}</span>
                <span className={hot ? "text-status-danger" : ""}>{line.message}</span>
              </li>
            );
          })}
        </ul>
      </TabsContent>
      <TabsContent value="traces" className="space-y-4 p-3">
        {evidence.traces.length === 0 ? (
          <EmptyBlock title="No traces" detail="No trace was captured for this incident." />
        ) : (
          evidence.traces.map((trace) => {
            const max = Math.max(...trace.spans.map((span) => span.durationMs), 1);
            return (
              <div key={trace.id}>
                <div className="mb-2 font-mono text-[11px] text-muted-foreground">
                  {trace.traceId} · {trace.clock}
                </div>
                <ol className="space-y-2">
                  {trace.spans.map((span, index) => (
                    <li key={span.id} style={{ marginLeft: index * 16 }}>
                      <div className="flex items-baseline justify-between gap-3 text-xs">
                        <span>
                          <span className="font-medium">{span.service}</span>
                          <span className="ml-2 font-mono text-muted-foreground">{span.name}</span>
                        </span>
                        <span className={`font-mono ${span.slow ? "text-status-danger" : ""}`}>{formatLatency(span.durationMs)}</span>
                      </div>
                      <div className="mt-1 h-1.5 bg-muted">
                        <div
                          className={span.slow ? "h-full bg-status-danger" : "h-full bg-status-info"}
                          style={{ width: `${Math.max(4, (span.durationMs / max) * 100)}%` }}
                        />
                      </div>
                      {span.slow ? <div className="mt-1 text-[11px] text-status-danger">Slow span</div> : null}
                    </li>
                  ))}
                </ol>
              </div>
            );
          })
        )}
      </TabsContent>
      <TabsContent value="events">
        <Table>
          <TableHeader>
            <TableRow>
              {["Time", "Type", "Reason", "Object", "Message"].map((heading) => (
                <TableHead key={heading} className="h-8 text-[11px] uppercase tracking-wide text-muted-foreground">
                  {heading}
                </TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {evidence.kubernetesEvents.map((event, index) => (
              <TableRow key={`${event.reason}-${index}`}>
                <TableCell className="font-mono text-xs">{event.clock}</TableCell>
                <TableCell>
                  <StatusBadge value={event.type} />
                </TableCell>
                <TableCell>{event.reason}</TableCell>
                <TableCell className="font-mono text-xs">{event.object}</TableCell>
                <TableCell className="whitespace-normal">{event.message}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TabsContent>
      <TabsContent value="deployments">
        <Table>
          <TableHeader>
            <TableRow>
              {["Time", "Version", "Status", "Actor", "Change"].map((heading) => (
                <TableHead key={heading} className="h-8 text-[11px] uppercase tracking-wide text-muted-foreground">
                  {heading}
                </TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {evidence.deployments.map((deployment) => (
              <TableRow key={deployment.id}>
                <TableCell className="font-mono text-xs">
                  {deployment.clock}
                  <div className="text-muted-foreground">{formatAgo(deployment.at)}</div>
                </TableCell>
                <TableCell className="font-mono">
                  {deployment.version}
                  <div className="text-[11px] text-muted-foreground">from {deployment.previousVersion}</div>
                </TableCell>
                <TableCell>
                  <StatusBadge value={deployment.status} />
                </TableCell>
                <TableCell>{deployment.actor}</TableCell>
                <TableCell className="whitespace-normal text-muted-foreground">{deployment.change}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TabsContent>
    </Tabs>
  );
}
