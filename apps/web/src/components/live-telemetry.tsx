"use client";

import { useState } from "react";
import { StatusBadge } from "@/components/status-badge";
import { Panel } from "@/components/states";
import { formatAgo, formatLatency, formatPercent } from "@/lib/format";
import { useApi } from "@/lib/use-api";
import type { TelemetrySnapshot, TraceDetail, TraceList } from "@/lib/types";

export function LiveReliability({ serviceId }: { serviceId: string }) {
  const telemetry = useApi<TelemetrySnapshot>(`/api/v1/services/${serviceId}/telemetry`, 5000);
  const traces = useApi<TraceList>(`/api/v1/services/${serviceId}/traces`, 5000);
  const snap = telemetry.data;
  const available = Boolean(snap?.available);

  return (
    <div className="space-y-3">
      <Panel title="Live reliability" action={<span className="text-[11px] text-muted-foreground">Source: {available ? "Prometheus" : "unavailable"}</span>}>
        {!snap || !available ? (
          <p className="text-sm text-muted-foreground">{snap?.message || telemetry.error || "Telemetry unavailable"}</p>
        ) : (
          <>
            <dl className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
              <Metric label="Request rate" value={`${snap.requestRate.toFixed(2)} req/s`} />
              <Metric label="Error rate" value={formatPercent(snap.errorRate * 100)} />
              <Metric label="P95 latency" value={formatLatency(snap.p95LatencyMs)} />
              <Metric label="Availability" value={formatPercent(snap.availability * 100)} />
              <Metric label="Recent request volume" value={snap.requests.toFixed(0)} />
              <Metric label="P50 latency" value={formatLatency(snap.p50LatencyMs)} />
              <Metric label="P99 latency" value={formatLatency(snap.p99LatencyMs)} />
              <Metric label="Telemetry last updated" value={snap.updated ? formatAgo(snap.updated) : "—"} />
            </dl>
            <p className="mt-3 text-[11px] text-muted-foreground">Data source: {snap.dataSource}. Window: 1 minute. These numbers are not the seeded INC-142 series.</p>
          </>
        )}
      </Panel>
      {snap?.slo ? (
        <Panel title="Payment API SLO" action={<span className="text-[11px] text-muted-foreground">{snap.slo.label}</span>}>
          {!snap.slo.sufficient ? (
            <p className="text-sm text-muted-foreground">Not enough requests in the local observation window to calculate a budget.</p>
          ) : (
            <dl className="grid gap-3 sm:grid-cols-2 lg:grid-cols-5">
              <Metric label="Target" value={formatPercent(snap.slo.target, 1)} />
              <Metric label="Current window" value={snap.slo.window} />
              <Metric label="Availability" value={formatPercent(snap.slo.availability * 100)} />
              <Metric label="Error budget consumed" value={formatPercent(snap.slo.errorBudgetConsumed * 100, 0)} />
              <Metric label="Burn rate" value={`${snap.slo.burnRate.toFixed(1)}x`} />
            </dl>
          )}
          <p className="mt-3 text-[11px] text-muted-foreground">Local observation window. This is not a 30-day reliability claim. Source: Prometheus.</p>
        </Panel>
      ) : null}
      <Panel title="Recent traces" action={<span className="text-[11px] text-muted-foreground">Source: {traces.data?.source === "opentelemetry" ? "OpenTelemetry" : "unavailable"}</span>}>
        {traces.data?.source !== "opentelemetry" ? (
          <p className="text-sm text-muted-foreground">{traces.data?.message || traces.error || "Telemetry unavailable"}</p>
        ) : traces.data.traces.length === 0 ? (
          <p className="text-sm text-muted-foreground">No traces in the last 15 minutes.</p>
        ) : (
          <TraceTable traces={traces.data.traces} />
        )}
      </Panel>
    </div>
  );
}

function TraceTable({ traces }: { traces: TraceList["traces"] }) {
  const [selected, setSelected] = useState<string | null>(null);
  const visible = traces.slice(0, 8);
  return (
    <div>
      {selected ? <TraceTree id={selected} /> : <p className="mb-2 text-xs text-muted-foreground">Select a trace to open the span tree.</p>}
      <ul>
        {visible.map((trace) => (
          <li key={trace.id} className="border-b border-border py-2 last:border-b-0">
            <button
              type="button"
              className={`flex w-full cursor-pointer flex-wrap items-center gap-2 text-left ${selected === trace.id ? "bg-muted" : ""}`}
              onClick={() => setSelected(trace.id)}
            >
              <span className="font-mono text-xs">{trace.id}</span>
              <span className="text-sm">{trace.rootService}</span>
              <span className="font-mono text-xs">{formatLatency(trace.durationMs)}</span>
              <span className="text-xs text-muted-foreground">{trace.spans} spans</span>
              <StatusBadge value={trace.status === "error" ? "critical" : "healthy"} />
              <span className="ml-auto text-xs text-muted-foreground">{formatAgo(trace.start)}</span>
            </button>
          </li>
        ))}
      </ul>
    </div>
  );
}

function TraceTree({ id }: { id: string }) {
  const detail = useApi<TraceDetail>(`/api/v1/traces/${id}`);
  if (!detail.data) {
    return <p className="mt-3 text-sm text-muted-foreground">{detail.error || "Loading trace"}</p>;
  }
  return (
    <div className="mt-3 border border-border bg-muted/40 p-3">
      <div className="mb-2 text-xs text-muted-foreground">
        {detail.data.rootService} · {formatLatency(detail.data.durationMs)} · source {detail.data.source}
      </div>
      <ul className="space-y-1">
        {detail.data.tree.map((span, index) => (
          <li key={`${span.operation}-${index}`} style={{ paddingLeft: span.depth * 16 }} className="text-sm">
            <span className="font-medium">{span.service}</span>
            <span className="mx-2 font-mono text-xs text-muted-foreground">{span.operation}</span>
            <span className="font-mono text-xs">{formatLatency(span.durationMs)}</span>
            <span className="ml-2 text-xs uppercase text-muted-foreground">{span.status}</span>
            {span.attributes
              ? Object.entries(span.attributes).map(([key, value]) => (
                  <span key={key} className="ml-2 font-mono text-[11px] text-muted-foreground">
                    {key}={value}
                  </span>
                ))
              : null}
          </li>
        ))}
      </ul>
    </div>
  );
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <div className="text-[11px] uppercase tracking-wide text-muted-foreground">{label}</div>
      <div className="mt-1 font-mono text-sm">{value}</div>
    </div>
  );
}
