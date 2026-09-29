"use client";

import { useState } from "react";
import { StatusBadge } from "@/components/status-badge";
import { EmptyBlock, ErrorBlock, LoadingBlock, PageHeader, Panel } from "@/components/states";
import { Button } from "@/components/ui/button";
import { apiPost } from "@/lib/api";
import { formatLatency, formatPercent } from "@/lib/format";
import { useApi } from "@/lib/use-api";

interface Rollout {
  id: string;
  service: string;
  namespace: string;
  cluster: string;
  stableVersion: string;
  candidateVersion: string;
  state: string;
  weight: number;
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

export function RolloutsPage() {
  const list = useApi<Rollout[]>("/api/v1/rollouts", 3000);
  const [pending, setPending] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);
  if (list.loading && !list.data) return <LoadingBlock />;
  if (!list.data) return <ErrorBlock message={list.error ?? "Rollouts unavailable"} onRetry={() => void list.reload()} />;
  const active = list.data.find((item) => !["SUCCEEDED", "FAILED", "ABORTED"].includes(item.state)) ?? list.data[0];

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
  return (
    <div className="grid gap-3">
      <Panel title={`${view.id} · ${view.service}`}>
        <dl className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          <Field label="Status" value={view.state} />
          <Field label="Stable" value={view.stableVersion} />
          <Field label="Candidate" value={view.candidateVersion} />
          <Field label="Traffic to candidate" value={`${view.weight}%`} />
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
            <Field label="Stable error / p95" value={`${view.stableErrorRate ? formatPercent(view.stableErrorRate * 100) : "—"} / ${view.stableP95 ? formatLatency(view.stableP95 * 1000) : "—"}`} />
          </dl>
          <p className="mt-3 text-sm">Proposal: <span className="font-mono">{view.proposalAction || "none yet"}</span></p>
          {view.state === "AWAITING_APPROVAL" && view.proposalAction ? (
            <div className="mt-3">
              <Button disabled={pending} onClick={() => onApprove(view.proposalAction)}>
                {pending ? "Recording approval…" : "Approve constrained action"}
              </Button>
            </div>
          ) : null}
          {actionError ? <p className="mt-2 text-xs text-red-300">{actionError}</p> : null}
          {view.verification ? <p className="mt-3 text-sm text-muted-foreground">Verification: {view.verification}</p> : null}
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

function Field({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <div className="text-[11px] uppercase tracking-wide text-muted-foreground">{label}</div>
      <div className="mt-1 font-mono text-sm">{value}</div>
    </div>
  );
}
