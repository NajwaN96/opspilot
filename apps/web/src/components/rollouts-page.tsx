"use client";

import { useState } from "react";
import { StatusBadge } from "@/components/status-badge";
import { EmptyBlock, ErrorBlock, LoadingBlock, PageHeader, Panel } from "@/components/states";
import { Button } from "@/components/ui/button";
import { apiPost } from "@/lib/api";
import { isPortfolioRuntime } from "@/lib/runtime";
import { formatLatency, formatPercent } from "@/lib/format";
import { useApi } from "@/lib/use-api";

interface Rollout {
  id: string;
  service: string;
  namespace: string;
  cluster: string;
  stableVersion: string;
  liveStableVersion?: string;
  liveStableKnown?: boolean;
  liveStableReady?: boolean;
  candidateVersion: string;
  state: string;
  weight: number;
  stageWeight: number;
  liveWeight: number;
  liveWeightKnown: boolean;
  candidateActivity?: string;
  incidentId?: string;
  incidentStatus?: string;
  alerts?: { name: string; alertmanager: string; interpretation: string; note?: string }[];
  analysis: string;
  proposalAction: string;
  aiAction: string;
  aiSummary: string;
  aiStatus: string;
  aiMismatch: boolean;
  verification: string;
  requests: number;
  errorRate: number;
  p95: number;
  stableErrorRate: number;
  stableP95: number;
  candidateReady: boolean;
  stageStartedAt?: string;
  events?: { at: string; title: string; detail: string; kind: string }[];
}

const terminalStates = ["SUCCEEDED", "FAILED", "ABORTED", "NEEDS_ATTENTION"];

export function RolloutsPage() {
  const list = useApi<Rollout[]>("/api/v1/rollouts", 3000);
  const [pending, setPending] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);
  const [selected, setSelected] = useState<string | null>(null);
  if (list.loading && !list.data) return <LoadingBlock />;
  if (!list.data) return <ErrorBlock message={list.error ?? "Rollouts unavailable"} onRetry={() => void list.reload()} />;
  const active =
    list.data.find((item) => item.id === selected) ??
    list.data.find((item) => !["SUCCEEDED", "FAILED", "ABORTED"].includes(item.state)) ??
    list.data[0];

  async function approve(action: string) {
    if (!active) return;
    setPending(true);
    setActionError(null);
    try {
      await apiPost(`/api/v1/rollouts/${active.id}/approve`, { action });
      await list.reload();
    } catch (err) {
      setActionError(err instanceof Error ? err.message : "Approval failed");
    } finally {
      setPending(false);
    }
  }

  return (
    <div>
      <PageHeader
        kicker="demo-shop"
        title="Progressive delivery"
        description="A canary runs beside the stable payment-api. Prometheus decides pass or fail. A person approves the only production change."
      />
      {list.data.length > 0 ? (
        <div className="mb-3 flex gap-2 overflow-x-auto pb-1">
          {list.data.map((item) => (
            <button
              key={item.id}
              type="button"
              onClick={() => setSelected(item.id)}
              className={`shrink-0 rounded-md border px-3 py-2 text-left text-xs ${item.id === active?.id ? "border-foreground/40 bg-card" : "border-border text-muted-foreground"}`}
            >
              <div className="font-mono text-foreground">{item.id}</div>
              <div>
                {item.state} · stage {item.stageWeight || item.weight}% · live {item.liveWeightKnown ? `${item.liveWeight}%` : "unknown"}
              </div>
            </button>
          ))}
        </div>
      ) : null}
      {!active ? (
        <EmptyBlock
          title="No canary is running"
          detail="Start a healthy 1.5.0 canary or a 1.6.0-bad canary from Reliability Lab. Those controls are local experiments."
        />
      ) : (
        <RolloutDetail item={active} pending={pending} actionError={actionError} onApprove={approve} />
      )}
    </div>
  );
}

function RolloutDetail({
  item,
  pending,
  actionError,
  onApprove,
}: {
  item: Rollout;
  pending: boolean;
  actionError: string | null;
  onApprove: (action: string) => void;
}) {
  const detail = useApi<Rollout>(`/api/v1/rollouts/${item.id}`, 3000);
  const view = detail.data ?? item;
  const events = view.events ?? [];
  const gateKnown = view.analysis === "PASS" || view.analysis === "FAIL" || view.analysis === "INSUFFICIENT_DATA";
  const terminal = terminalStates.includes(view.state);
  const stage = view.stageWeight || view.weight;
  const approved = events.some((event) => event.title === "Human approved");
  const executed = events.some((event) => event.title === "Candidate removed" || event.title === "Candidate promoted");
  const liveStable = view.liveStableKnown
    ? `${view.liveStableVersion}${view.liveStableReady ? " — healthy" : " — not ready"}`
    : "unknown";
  return (
    <div className="grid gap-3">
      <Panel title={`${view.id} · ${view.service}`}>
        <dl className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          <Field label="Status" value={view.state} />
          <Field label={terminal ? "Last evaluated stage" : "Current stage"} value={`${stage}%`} />
          <Field label="Live canary traffic" value={view.liveWeightKnown ? `${view.liveWeight}%` : "unknown"} />
          <Field label="Candidate" value={`${view.candidateVersion} — ${activityLabel(view.candidateActivity)}`} />
          <Field label="Baseline" value={view.stableVersion} />
          <Field label="Live stable" value={liveStable} />
          {view.state === "SUCCEEDED" ? <Field label="Promoted version" value={view.candidateVersion} /> : null}
          <Field label="SLO gate" value={view.analysis || "waiting"} />
          <Field label="Remediation" value={approved && executed ? "approved + executed" : approved ? "approved" : "not executed"} />
          <Field label="Incident" value={view.incidentId ? `${view.incidentId} ${view.incidentStatus}` : "none"} />
        </dl>
      </Panel>
      <div className="grid gap-3 lg:grid-cols-2">
        <Panel title="Deterministic SLO gate">
          <p className="text-sm text-muted-foreground">Minimum 20 requests, error rate at most 5%, p95 at most 300ms, candidate Ready. Missing data is not a pass.</p>
          <dl className="mt-3 grid gap-3 sm:grid-cols-2">
            <Field label="Analysis" value={view.analysis || "waiting"} />
            <Field label="Candidate ready" value={view.candidateReady ? "yes" : "no"} />
            <Field label="Candidate requests" value={view.requests ? String(Math.round(view.requests)) : "—"} />
            <Field label="Candidate error" value={view.requests ? formatPercent(view.errorRate * 100) : "—"} />
            <Field label="Candidate p95" value={view.p95 ? formatLatency(view.p95 * 1000) : "—"} />
            <Field label="Stable error / p95" value={`${gateKnown ? formatPercent(view.stableErrorRate * 100) : "—"} / ${gateKnown ? formatLatency(view.stableP95 * 1000) : "—"}`} />
          </dl>
          <p className="mt-3 text-sm">Proposal: <span className="font-mono">{view.proposalAction || "none yet"}</span></p>
          {view.state === "AWAITING_APPROVAL" && view.proposalAction && !isPortfolioRuntime() ? (
            <div className="mt-3">
              <Button disabled={pending} onClick={() => onApprove(view.proposalAction)}>
                {pending ? "Recording approval…" : "Approve constrained action"}
              </Button>
            </div>
          ) : null}
          {isPortfolioRuntime() && view.state === "AWAITING_APPROVAL" ? (
            <p className="mt-3 text-xs text-muted-foreground">
              This canary is a sanitized sample. Approval stays on the local reliability lab and is not sent from AWS.
            </p>
          ) : null}
          {actionError ? <p className="mt-2 text-xs text-red-300">{actionError}</p> : null}
          {view.verification ? <p className="mt-3 text-sm text-muted-foreground">Verification: {view.verification}</p> : null}
          {view.alerts && view.alerts.length > 0 ? (
            <ul className="mt-3 grid gap-2">
              {view.alerts.filter((alert) => alert.alertmanager === "firing" || alert.interpretation === "recovering").map((alert) => (
                <li key={alert.name} className="text-xs text-muted-foreground">
                  <span className="font-mono text-foreground">{alert.name}</span> {alert.alertmanager} · {alert.interpretation}
                  {alert.note ? ` — ${alert.note}` : ""}
                </li>
              ))}
            </ul>
          ) : null}
        </Panel>
        <Panel title="AI investigator recommendation" action={<span className="text-[11px] uppercase tracking-wide text-sky-200">Not a gate</span>}>
          <p className="text-sm text-muted-foreground">The model may recommend continue, promote, or abort. A failing SLO gate still rejects promotion.</p>
          <dl className="mt-3 grid gap-3 sm:grid-cols-2">
            <Field label="Status" value={view.aiStatus || "not requested"} />
            <Field label="Recommended action" value={view.aiAction || "—"} />
          </dl>
          {view.aiSummary ? <p className="mt-3 text-sm">{view.aiSummary}</p> : null}
          {view.aiMismatch ? <p className="mt-3 text-sm text-amber-100">The model recommended promotion and the SLO gate failed. Promotion stays rejected.</p> : null}
        </Panel>
      </div>
      <Panel title="Timeline" padded={false}>
        {events.length === 0 ? <p className="px-3 py-3 text-sm text-muted-foreground">Events appear as the canary is observed.</p> : null}
        <ul>
          {events.map((event) => (
            <li key={`${event.at}-${event.title}`} className="border-b border-border px-3 py-2">
              <div className="flex flex-wrap items-center gap-2">
                <StatusBadge value={event.kind} />
                <span className="text-sm">{event.title}</span>
              </div>
              <p className="mt-1 text-xs text-muted-foreground">{event.detail}</p>
            </li>
          ))}
        </ul>
      </Panel>
    </div>
  );
}

function activityLabel(value?: string): string {
  if (value === "idle") return "inactive";
  if (value === "ready") return "ready";
  if (value === "unavailable") return "unavailable";
  return "unknown";
}

function Field({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <div className="text-[11px] uppercase tracking-wide text-muted-foreground">{label}</div>
      <div className="mt-1 font-mono text-sm">{value}</div>
    </div>
  );
}
