"use client";

import { useState } from "react";
import { StatusBadge } from "@/components/status-badge";
import { Panel } from "@/components/states";
import { Button } from "@/components/ui/button";
import { apiPost } from "@/lib/api";
import { formatClock } from "@/lib/format";
import { useApi } from "@/lib/use-api";

interface EvidenceItem {
  id: string;
  source: string;
  kind: string;
  at?: string;
  title: string;
  body: string;
}

interface Cause {
  cause: string;
  confidence: number;
  supporting_evidence_ids: string[];
  contradicting_evidence_ids: string[];
}

interface InvestigationView {
  status: string;
  provider: string;
  model: string;
  real: boolean;
  snapshotId?: string;
  snapshotVersion: number;
  summary?: string;
  likelyCauses?: Cause[];
  observations?: { statement: string; evidence_ids: string[] }[];
  recommendedAction: { action_type: string; reason: string; evidence_ids: string[] };
  missingInformation?: string[];
  riskNotes?: string[];
  validation: { accepted: boolean; errors?: string[] };
  evidence?: EvidenceItem[];
  omitted?: string[];
  error?: string;
  startedAt?: string;
  completedAt?: string;
}

export function AIInvestigator({ incidentId }: { incidentId: string }) {
  const query = useApi<InvestigationView>(`/api/v1/incidents/${incidentId}/investigation`, 3000);
  const [pending, setPending] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);
  const [selected, setSelected] = useState<string | null>(null);
  const view = query.data;
  const evidence = new Map((view?.evidence ?? []).map((item) => [item.id, item]));
  const chosen = selected ? evidence.get(selected) : undefined;
  const canRun = !view || ["not_requested", "failed", "invalid_output", "unavailable"].includes(view.status);

  async function run() {
    setPending(true);
    setActionError(null);
    try {
      await apiPost<InvestigationView>(`/api/v1/incidents/${incidentId}/investigation`, {});
      await query.reload();
    } catch (err) {
      setActionError(err instanceof Error ? err.message : "Investigation failed");
    } finally {
      setPending(false);
    }
  }

  return (
    <Panel
      title="AI Investigator"
      action={<span className="text-[11px] uppercase tracking-wide text-sky-200">{view?.real ? "Real OpenAI" : "Not an executor"}</span>}
    >
      <p className="text-sm text-muted-foreground">
        The investigator reads a prepared evidence snapshot. A recommendation is not a proposal and does not change Kubernetes.
      </p>
      <dl className="mt-3 grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <Field label="Status" value={view?.status ?? "loading"} />
        <Field label="Provider" value={view?.provider || "—"} />
        <Field label="Model" value={view?.model || "—"} />
        <Field label="Evidence snapshot" value={view?.snapshotVersion ? `v${view.snapshotVersion}` : "—"} />
      </dl>
      {view?.error || view?.status === "unavailable" ? (
        <p className="mt-3 text-sm text-amber-100">
          AI Investigator Unavailable. {view?.error?.startsWith("quota") ? "The OpenAI account has no remaining credits, so this result is not a model investigation. Detection and approval still run." : view?.error}
        </p>
      ) : null}
      {view?.validation && !view.validation.accepted && (view.validation.errors ?? []).length > 0 ? (
        <p className="mt-3 text-sm text-amber-100">Output was rejected: {view.validation.errors?.join("; ")}</p>
      ) : null}
      {view?.summary ? (
        <div className="mt-4">
          <h3 className="text-sm font-medium">AI synthesis</h3>
          <p className="mt-1 text-sm">{view.summary}</p>
        </div>
      ) : null}
      {(view?.likelyCauses ?? []).map((cause) => (
        <div key={cause.cause} className="mt-4">
          <h3 className="text-sm font-medium">{cause.cause}</h3>
          <p className="mt-1 text-xs text-muted-foreground">Model confidence {Math.round(cause.confidence * 100)}%. This is not a calibrated probability.</p>
          <Cite label="Supporting evidence" ids={cause.supporting_evidence_ids} onSelect={setSelected} />
          <Cite label="Contradicting evidence" ids={cause.contradicting_evidence_ids} onSelect={setSelected} />
        </div>
      ))}
      {view?.recommendedAction?.action_type ? (
        <div className="mt-4 border border-border p-3">
          <div className="text-[11px] uppercase tracking-wide text-muted-foreground">AI recommended action</div>
          <div className="mt-1 font-mono text-sm">{view.recommendedAction.action_type}</div>
          <p className="mt-1 text-sm text-muted-foreground">{view.recommendedAction.reason}</p>
          <Cite label="Cited evidence" ids={view.recommendedAction.evidence_ids} onSelect={setSelected} />
        </div>
      ) : null}
      {(view?.missingInformation ?? []).length > 0 ? (
        <p className="mt-3 text-sm text-muted-foreground">Missing: {view?.missingInformation?.join("; ")}</p>
      ) : null}
      {(view?.riskNotes ?? []).length > 0 ? <p className="mt-2 text-sm text-muted-foreground">{view?.riskNotes?.join(" ")}</p> : null}
      {(view?.omitted ?? []).length > 0 ? <p className="mt-2 text-xs text-muted-foreground">Omitted: {view?.omitted?.join(", ")}</p> : null}
      {chosen ? (
        <div className="mt-4 border border-sky-400/30 bg-sky-400/5 p-3">
          <div className="flex flex-wrap items-center gap-2">
            <span className="font-mono text-sm">{chosen.id}</span>
            <StatusBadge value={chosen.source} />
          </div>
          <p className="mt-1 text-sm font-medium">{chosen.title}</p>
          <p className="mt-1 text-sm text-muted-foreground">{chosen.body}</p>
          {chosen.at ? <p className="mt-1 font-mono text-[11px] text-muted-foreground">{formatClock(chosen.at)}</p> : null}
        </div>
      ) : null}
      <div className="mt-3">
        {canRun ? (
          <Button disabled={pending} onClick={() => void run()}>
            {pending ? "Investigating…" : "Run AI Investigation"}
          </Button>
        ) : null}
        {actionError ? <p className="mt-2 text-xs text-red-300">{actionError}</p> : null}
      </div>
    </Panel>
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

function Cite({ label, ids, onSelect }: { label: string; ids: string[]; onSelect: (id: string) => void }) {
  if (!ids || ids.length === 0) return null;
  return (
    <div className="mt-2">
      <div className="text-[11px] uppercase tracking-wide text-muted-foreground">{label}</div>
      <div className="mt-1 flex flex-wrap gap-1">
        {ids.map((id) => (
          <button key={id} type="button" className="border border-sky-400/40 px-2 py-0.5 font-mono text-xs text-sky-100" onClick={() => onSelect(id)}>
            {id}
          </button>
        ))}
      </div>
    </div>
  );
}
