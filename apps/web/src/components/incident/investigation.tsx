"use client";

import Link from "next/link";
import { useState } from "react";
import { Check, Circle } from "lucide-react";
import { EvidencePanel } from "@/components/incident/evidence";
import { StatusBadge } from "@/components/status-badge";
import { ErrorBlock, LoadingBlock, Panel } from "@/components/states";
import { Button } from "@/components/ui/button";
import { Progress } from "@/components/ui/progress";
import { apiPost } from "@/lib/api";
import { completedSteps, formatClock, formatDuration, formatLatency, formatPercent } from "@/lib/format";
import { useApi } from "@/lib/use-api";
import { useTween } from "@/lib/use-tween";
import type { Incident, ReadyStatus, Remediation } from "@/lib/types";

export function Investigation({ id }: { id: string }) {
  const incidentQuery = useApi<Incident>(`/api/v1/incidents/${id}`, 1000);
  const ready = useApi<ReadyStatus>("/ready", 15000);
  const [pending, setPending] = useState(false);
  const [resetting, setResetting] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);

  if (incidentQuery.loading && !incidentQuery.data) return <LoadingBlock label="Loading incident" />;
  if (!incidentQuery.data) {
    return <ErrorBlock message={incidentQuery.error ?? "Incident not found"} onRetry={() => void incidentQuery.reload()} />;
  }

  const incident = incidentQuery.data;
  const done = completedSteps(incident.remediation?.steps);
  const total = incident.remediation?.steps.length ?? 5;

  async function approve() {
    setPending(true);
    setActionError(null);
    try {
      await apiPost<Remediation>(`/api/v1/incidents/${id}/remediations`, { action: "rollback" });
      await incidentQuery.reload();
    } catch (err) {
      setActionError(err instanceof Error ? err.message : "Remediation failed");
    } finally {
      setPending(false);
    }
  }

  async function resetDemo() {
    setResetting(true);
    setActionError(null);
    try {
      await apiPost<{ status: string }>("/api/v1/demo/reset", {});
      await incidentQuery.reload();
    } catch (err) {
      setActionError(err instanceof Error ? err.message : "Reset failed");
    } finally {
      setResetting(false);
    }
  }

  return (
    <div>
      <div className="mb-3 flex flex-wrap items-start justify-between gap-3">
        <div>
          <div className="flex flex-wrap items-center gap-2">
            <h1 className="font-mono text-lg font-semibold">{incident.id}</h1>
            <StatusBadge value={incident.severity} />
            <span className="text-[11px] uppercase tracking-wide text-muted-foreground">Source: Simulation</span>
            <StatusBadge value={incident.status} />
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
            {incident.snapshot.version ? ` · ${incident.snapshot.version}` : ""}
          </p>
        </div>
      </div>
      <p className="mb-3 border border-border bg-card px-3 py-2 text-xs text-muted-foreground">
        Simulated analysis for the current MVP. Approving a rollback records the decision and runs a fake executor. kubectl is not called.
      </p>
      <div className="mb-3 grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <LiveMetric label="Error rate" value={incident.snapshot.errorRate} render={(value) => formatPercent(value, 1)} />
        <LiveMetric label="P95 latency" value={incident.snapshot.p95LatencyMs} render={(value) => formatLatency(value)} />
        <LiveMetric label="Database connections" value={incident.snapshot.dbConnections} render={(value) => `${Math.round(value)}%`} />
        <div className="border border-border bg-card px-3 py-2">
          <div className="text-[11px] uppercase tracking-wide text-muted-foreground">Status</div>
          <div className="mt-1">
            <StatusBadge value={incident.status} />
          </div>
        </div>
      </div>
      <div className="grid gap-3 xl:grid-cols-[minmax(0,1.65fr)_minmax(300px,0.85fr)]">
        <div className="space-y-3">
          <Panel title="Timeline" padded={false}>
            <ol className="px-3 py-2">
              {(incident.timeline ?? []).map((event, index) => (
                <li key={`${event.at}-${event.title}-${index}`} className="grid grid-cols-[3.25rem_12px_minmax(0,1fr)] gap-2">
                  <time className="pt-1 font-mono text-xs text-muted-foreground">{event.clock}</time>
                  <div className="flex flex-col items-center">
                    <span className={`mt-1.5 size-2 rounded-full ${event.kind === "remediation" ? "bg-emerald-400" : "bg-amber-300"}`} />
                    {index < (incident.timeline?.length ?? 0) - 1 ? <span className="w-px flex-1 bg-border" /> : null}
                  </div>
                  <div className="pb-3">
                    <div className="text-sm font-medium">{event.title}</div>
                    <div className="text-xs text-muted-foreground">{event.detail}</div>
                  </div>
                </li>
              ))}
            </ol>
          </Panel>
          <Panel title="Evidence" padded={false}>
            {incident.evidence ? <EvidencePanel evidence={incident.evidence} /> : <p className="p-3 text-sm text-muted-foreground">No evidence attached.</p>}
          </Panel>
        </div>
        <div className="space-y-3">
          <Panel title="Root cause analysis" action={incident.analysis?.simulated ? <StatusBadge value="simulated" /> : null}>
            <p className="text-sm font-medium">{incident.analysis?.cause ?? "No analysis recorded."}</p>
            {incident.analysis ? (
              <>
                <div className="mt-3">
                  <div className="flex items-baseline justify-between text-xs text-muted-foreground">
                    <span>Confidence</span>
                    <span className="font-mono text-sm text-foreground">{Math.round(incident.analysis.confidence * 100)}%</span>
                  </div>
                  <Progress className="mt-1" value={Math.round(incident.analysis.confidence * 100)} />
                </div>
                <ul className="mt-3 space-y-1.5 text-sm">
                  {incident.analysis.evidence.map((item) => (
                    <li key={item} className="flex gap-2">
                      <span className="mt-2 size-1 shrink-0 bg-muted-foreground" aria-hidden />
                      <span>{item}</span>
                    </li>
                  ))}
                </ul>
                <p className="mt-3 text-[11px] text-muted-foreground">
                  This probable cause is simulated. A future investigator may propose it, but it will not receive Kubernetes admin access.
                </p>
              </>
            ) : null}
          </Panel>
          <Panel title="Remediation">
            {incident.recommendation ? (
              <dl className="grid grid-cols-2 gap-2 text-sm">
                <div className="col-span-2">
                  <dt className="text-[11px] uppercase tracking-wide text-muted-foreground">Recommended action</dt>
                  <dd className="font-medium">{incident.recommendation.summary}</dd>
                </div>
                <Field label="From" value={incident.recommendation.from || "—"} />
                <Field label="To" value={incident.recommendation.to || "—"} />
                <Field label="Risk" value={incident.recommendation.risk} />
                <Field label="Policy" value={incident.recommendation.policy} />
              </dl>
            ) : (
              <p className="text-sm text-muted-foreground">No recommendation.</p>
            )}
            {incident.recommendation?.allowed && !incident.remediation ? (
              <Button className="mt-3 w-full" onClick={() => void approve()} disabled={pending}>
                {pending ? "Recording approval…" : "Approve & Execute"}
              </Button>
            ) : null}
            {actionError ? <p className="mt-2 text-xs text-red-300">{actionError}</p> : null}
            {incident.remediation ? (
              <div className="mt-3">
                <div className="mb-2 flex items-center justify-between text-[11px] uppercase tracking-wide text-muted-foreground">
                  <span>{incident.remediation.simulated ? "Simulated executor" : "Executor"}</span>
                  <span className="font-mono normal-case">
                    {done}/{total}
                  </span>
                </div>
                <Progress value={(done / total) * 100} />
                <ol className="mt-3 space-y-2">
                  {incident.remediation.steps.map((step) => (
                    <li key={step.name} className="flex items-start gap-2 text-sm">
                      {step.status === "complete" ? (
                        <Check className="mt-0.5 size-4 text-emerald-300" aria-hidden />
                      ) : (
                        <Circle className={`mt-0.5 size-4 ${step.status === "active" ? "text-amber-200" : "text-muted-foreground"}`} aria-hidden />
                      )}
                      <span>
                        <span className={step.status === "pending" ? "text-muted-foreground" : ""}>{step.name}</span>
                        {step.clock ? <span className="ml-2 font-mono text-[11px] text-muted-foreground">{step.clock}</span> : null}
                      </span>
                    </li>
                  ))}
                </ol>
                {incident.remediation.status === "succeeded" ? (
                  <p className="mt-3 text-sm text-emerald-300">Incident resolved. payment-api is serving {incident.snapshot.version}.</p>
                ) : null}
              </div>
            ) : null}
            {ready.data?.demoResetEnabled && incident.id === "INC-142" ? (
              <Button className="mt-3 w-full" variant="outline" onClick={() => void resetDemo()} disabled={resetting}>
                {resetting ? "Resetting demo…" : "Reset Demo"}
              </Button>
            ) : null}
            {ready.data?.demoResetEnabled && incident.id === "INC-142" ? (
              <p className="mt-2 text-[11px] text-muted-foreground">Development only. Restores INC-142. Does not delete cluster or Kubernetes data.</p>
            ) : null}
          </Panel>
        </div>
      </div>
      <div className="mt-3">
        <Panel title="Audit trail" action={<span className="text-[11px] text-muted-foreground">Source: PostgreSQL</span>} padded={false}>
          {!incident.audit || incident.audit.length === 0 ? (
            <p className="px-3 py-4 text-sm text-muted-foreground">No remediation actions recorded yet.</p>
          ) : (
            <ul>
              {incident.audit.map((record) => (
                <li key={record.id} className="border-b border-border px-3 py-3 last:border-b-0">
                  <div className="flex flex-wrap items-center gap-2">
                    <span className="font-mono text-xs text-muted-foreground">{formatClock(record.at)}</span>
                    <span className="font-medium">{record.action}</span>
                    <span className="text-xs text-muted-foreground">{record.actor}</span>
                  </div>
                  <p className="mt-1 text-sm text-muted-foreground">{record.detail}</p>
                  <p className="mt-1 font-mono text-[11px] text-muted-foreground">
                    policy {record.policyResult || "—"} · approval {record.approvalResult || "—"} · execution {record.executionStatus || "—"} · verification {record.verificationResult || "—"}
                  </p>
                </li>
              ))}
            </ul>
          )}
        </Panel>
      </div>
    </div>
  );
}

function LiveMetric({ label, value, render }: { label: string; value: number; render: (value: number) => string }) {
  const tweened = useTween(value);
  return (
    <div className="border border-border bg-card px-3 py-2">
      <div className="text-[11px] uppercase tracking-wide text-muted-foreground">{label}</div>
      <div className="mt-1 font-mono text-xl tabular-nums">{render(tweened)}</div>
    </div>
  );
}

function Field({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt className="text-[11px] uppercase tracking-wide text-muted-foreground">{label}</dt>
      <dd className="font-mono text-sm">{value}</dd>
    </div>
  );
}
