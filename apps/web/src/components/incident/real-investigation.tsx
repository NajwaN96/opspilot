"use client";

import Link from "next/link";
import { useState } from "react";
import { AIInvestigator } from "@/components/incident/ai-investigator";
import { IncidentLifecycle } from "@/components/incident/lifecycle";
import { StatusBadge } from "@/components/status-badge";
import { Panel } from "@/components/states";
import { Button } from "@/components/ui/button";
import { apiPost } from "@/lib/api";
import { formatClock, formatDuration, formatLatency, formatPercent } from "@/lib/format";
import { useApi } from "@/lib/use-api";
import type { Experiment, ExperimentCatalog, Incident, Remediation } from "@/lib/types";

export function RealInvestigation({ incident, onReload }: { incident: Incident; onReload: () => Promise<void> }) {
  const experiments = useApi<ExperimentCatalog>("/api/v1/experiments", 2000);
  const [pending, setPending] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);
  const running = experiments.data?.realRuns?.find((run) => run.status === "running");
  const recovered = Boolean(incident.telemetryRecoveredAt) && incident.status !== "resolved";
  const rollback = incident.recommendation?.action === "rollback-payment-api" && incident.recommendation.allowed;
  const busy = incident.remediation && ["rolling", "verifying", "succeeded"].includes(incident.remediation.status);

  async function approve() {
    setPending(true);
    setActionError(null);
    try {
      await apiPost<Remediation>(`/api/v1/incidents/${incident.id}/remediations`, { action: "rollback-payment-api" });
      await onReload();
    } catch (err) {
      setActionError(err instanceof Error ? err.message : "Rollback failed");
    } finally {
      setPending(false);
    }
  }

  async function stop(run: Experiment) {
    setPending(true);
    setActionError(null);
    try {
      await apiPost<Experiment>(`/api/v1/experiments/${run.id}/stop`, {});
      await experiments.reload();
      await onReload();
    } catch (err) {
      setActionError(err instanceof Error ? err.message : "Stop failed");
    } finally {
      setPending(false);
    }
  }

  return (
    <div>
      <div className="mb-3">
        <div className="flex flex-wrap items-center gap-2">
          <h1 className="font-mono text-lg font-semibold">{incident.id}</h1>
          <StatusBadge value={incident.severity} />
          <StatusBadge value={incident.status} />
          <span className="border border-emerald-400/40 bg-emerald-400/10 px-2 py-0.5 text-[11px] uppercase tracking-wide text-emerald-200">
            Real evidence
          </span>
        </div>
        <p className="mt-1 text-base font-medium">{incident.title}</p>
        <p className="mt-1 text-sm text-muted-foreground">
          <Link className="hover:underline" href={`/services/${incident.serviceId}`}>
            {incident.serviceName}
          </Link>
          {" · "}
          {incident.cluster}
          {" · "}
          {formatDuration(incident.durationSec * 1000)}
        </p>
      </div>
      <IncidentLifecycle incident={incident} />
      <div className="mb-3 grid gap-2 border border-emerald-400/30 bg-card px-3 py-2 text-sm sm:grid-cols-2 xl:grid-cols-5">
        <Fact label="Incident source" value="Detection Engine" />
        <Fact label="Metrics source" value={incident.metricsSource || "Prometheus"} />
        <Fact label="Trace source" value={incident.traceSource || "OpenTelemetry"} />
        <Fact label="Kubernetes source" value={incident.kubernetesSource || "opspilot-dev"} />
        <Fact label="Simulation" value="No" />
      </div>
      {recovered ? (
        <div className="mb-3 border border-emerald-400/40 bg-emerald-400/10 px-3 py-2 text-sm text-emerald-100">
          Telemetry recovered at {formatClock(incident.telemetryRecoveredAt ?? "")}. The incident stays open for review. It was not closed automatically.
        </div>
      ) : null}
      <div className="grid gap-3 xl:grid-cols-2">
        <Panel title="Timeline">
          <ol className="space-y-3">
            {(incident.timeline ?? []).map((event) => (
              <li key={`${event.at}-${event.title}`}>
                <div className="font-mono text-[11px] text-muted-foreground">{event.clock || formatClock(event.at)}</div>
                <div className="text-sm font-medium">{event.title}</div>
                <p className="text-sm text-muted-foreground">{event.detail}</p>
              </li>
            ))}
          </ol>
        </Panel>
        <Panel title="Metrics evidence" action={<span className="text-[11px] text-emerald-200">Prometheus</span>}>
          <dl className="grid gap-3 sm:grid-cols-2">
            <Fact label="Error rate" value={formatPercent(incident.snapshot.errorRate)} mono />
            <Fact label="P95 latency" value={formatLatency(incident.snapshot.p95LatencyMs)} mono />
            <Fact label="Availability" value={formatPercent(incident.snapshot.availability)} mono />
            <Fact label="Request rate" value={`${(incident.evidence?.metrics?.[0]?.requestRate ?? 0).toFixed(2)} req/s`} mono />
          </dl>
          <p className="mt-3 text-xs text-muted-foreground">Captured when the rule opened. This is not the seeded INC-142 series.</p>
        </Panel>
        <Panel title="Detection rule" action={<span className="font-mono text-[11px]">{incident.ruleId}</span>}>
          <ul className="space-y-1 text-sm">
            {Object.entries(incident.thresholds ?? {}).map(([key, value]) => (
              <li key={key} className="flex justify-between gap-3 border-b border-border py-1">
                <span className="text-muted-foreground">{key}</span>
                <span className="font-mono">{String(value)}</span>
              </li>
            ))}
          </ul>
        </Panel>
        <Panel title="Deterministic findings">
          <p className="text-sm font-medium">Likely cause</p>
          <p className="mt-1 text-sm">{incident.analysis?.likelyCause || incident.analysis?.cause}</p>
          <p className="mt-3 text-sm font-medium">Supporting evidence</p>
          <ul className="mt-1 list-disc space-y-1 pl-4 text-sm text-muted-foreground">
            {(incident.analysis?.supporting ?? incident.analysis?.evidence ?? []).map((item) => (
              <li key={item}>{item}</li>
            ))}
          </ul>
          <p className="mt-3 text-sm font-medium">Contradicting evidence</p>
          <ul className="mt-1 list-disc space-y-1 pl-4 text-sm text-muted-foreground">
            {(incident.analysis?.contradicting ?? []).length === 0 ? <li>None recorded.</li> : null}
            {(incident.analysis?.contradicting ?? []).map((item) => (
              <li key={item}>{item}</li>
            ))}
          </ul>
        </Panel>
        <Panel title="Trace evidence" action={<span className="text-[11px] text-emerald-200">OpenTelemetry</span>}>
          {(incident.evidence?.traces ?? []).length === 0 ? (
            <p className="text-sm text-muted-foreground">No slow or failed traces were stored with this incident.</p>
          ) : (
            <ul className="space-y-2">
              {incident.evidence?.traces.map((trace) => (
                <li key={trace.traceId} className="text-sm">
                  <Link className="font-mono hover:underline" href={`/services/${incident.serviceId}`}>
                    {trace.traceId}
                  </Link>
                  <span className="ml-2 text-muted-foreground">
                    {trace.spans[0]?.service} · {formatLatency(trace.spans[0]?.durationMs ?? 0)} · {trace.spans[0]?.status}
                  </span>
                </li>
              ))}
            </ul>
          )}
        </Panel>
        <Panel title="Kubernetes state" action={<span className="text-[11px] text-emerald-200">opspilot-dev</span>}>
          <dl className="grid gap-3 sm:grid-cols-3">
            <Fact label="Version" value={incident.snapshot.version || "—"} mono />
            <Fact label="Status" value={incident.snapshot.status || "—"} />
            <Fact label="Service" value={incident.serviceName} mono />
          </dl>
          <ul className="mt-3 space-y-2 text-sm text-muted-foreground">
            {(incident.evidence?.kubernetesEvents ?? []).length === 0 ? <li>No recent events were stored.</li> : null}
            {(incident.evidence?.kubernetesEvents ?? []).map((event) => (
              <li key={`${event.clock}-${event.reason}`}>
                <span className="font-mono text-foreground">{event.reason}</span> {event.object}: {event.message}
              </li>
            ))}
          </ul>
        </Panel>
      </div>
      <div className="mt-3">
        <AIInvestigator incidentId={incident.id} />
      </div>
      <div className="mt-3 grid gap-3 xl:grid-cols-2">
        <Panel title="Deterministic proposal">
          <p className="text-sm">{incident.recommendation?.summary}</p>
          <p className="mt-2 text-sm text-muted-foreground">{incident.recommendation?.policy}</p>
          {rollback ? (
            <p className="mt-2 text-sm">
              Approval updates only <span className="font-mono">demo-shop/payment-api</span> from{" "}
              <span className="font-mono">{incident.recommendation?.from}</span> to{" "}
              <span className="font-mono">{incident.recommendation?.to}</span>. The image is chosen on the server.
            </p>
          ) : (
            <p className="mt-2 text-xs uppercase tracking-wide text-amber-200">Kubernetes rollback is not available for this incident.</p>
          )}
          <div className="mt-3 flex flex-wrap gap-2">
            {rollback && !busy ? (
              <Button disabled={pending} onClick={() => void approve()}>
                {pending ? "Rolling back…" : "Approve & Execute"}
              </Button>
            ) : null}
            <Button variant="outline" render={<Link href="/reliability-lab" />}>
              Open Reliability Lab
            </Button>
            {running ? (
              <Button variant="outline" disabled={pending} onClick={() => void stop(running)}>
                {pending ? "Stopping…" : "Stop Experiment"}
              </Button>
            ) : null}
          </div>
          {incident.remediation ? (
            <ol className="mt-3 space-y-2">
              {incident.remediation.steps.map((step) => (
                <li key={step.name} className="flex items-center justify-between gap-3 text-sm">
                  <span>{step.name}</span>
                  <StatusBadge value={step.status} />
                </li>
              ))}
            </ol>
          ) : null}
          {actionError ? <p className="mt-2 text-xs text-red-300">{actionError}</p> : null}
        </Panel>
        <Panel title="Audit trail">
          {(incident.audit ?? []).length === 0 ? <p className="text-sm text-muted-foreground">No audit events yet.</p> : null}
          <ul className="space-y-2">
            {(incident.audit ?? []).map((record) => (
              <li key={record.id} className="text-sm">
                <div className="font-mono text-[11px] text-muted-foreground">{formatClock(record.at)} · {record.actor}</div>
                <div>{record.action} · {record.executionStatus || record.approvalResult}</div>
                <p className="text-muted-foreground">{record.detail}</p>
              </li>
            ))}
          </ul>
        </Panel>
      </div>
    </div>
  );
}

function Fact({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div>
      <div className="text-[11px] uppercase tracking-wide text-muted-foreground">{label}</div>
      <div className={`mt-1 text-sm ${mono ? "font-mono" : ""}`}>{value}</div>
    </div>
  );
}
