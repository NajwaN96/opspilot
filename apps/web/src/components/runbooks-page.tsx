"use client";

import Link from "next/link";
import { ErrorBlock, LoadingBlock, PageHeader, Panel } from "@/components/states";
import type { Runbook } from "@/lib/platform";
import { useApi } from "@/lib/use-api";

export function RunbooksPage() {
  const library = useApi<{ runbooks: Runbook[] }>("/api/v1/platform/runbooks");
  if (library.loading && !library.data) return <LoadingBlock label="Loading runbooks" />;
  if (!library.data) return <ErrorBlock message={library.error ?? "Runbooks unavailable"} onRetry={() => void library.reload()} />;
  return (
    <div>
      <PageHeader
        kicker="Library"
        title="Runbooks"
        description="Symptoms, evidence, the allowed action, and how to verify it. A runbook does not grant the console permission to run it."
      />
      <Panel title="Scenarios" padded={false}>
        <ul>
          {library.data.runbooks.map((book) => (
            <li key={book.id} className="border-t border-border first:border-t-0">
              <Link href={`/runbooks/${book.id}`} className="block px-3 py-2 hover:bg-muted">
                <div className="font-medium">{book.title}</div>
                <div className="font-mono text-[11px] text-muted-foreground">{book.service}</div>
              </Link>
            </li>
          ))}
        </ul>
      </Panel>
    </div>
  );
}

export function RunbookDetail({ id }: { id: string }) {
  const book = useApi<Runbook>(`/api/v1/platform/runbooks/${id}`);
  if (book.loading && !book.data) return <LoadingBlock label="Loading runbook" />;
  if (!book.data?.title) return <ErrorBlock message={book.error ?? "Runbook not found"} onRetry={() => void book.reload()} />;
  const item = book.data;
  return (
    <div>
      <PageHeader kicker={item.service} title={item.title} description={item.detection} />
      <div className="grid gap-3 lg:grid-cols-2">
        <List title="Symptoms" items={item.symptoms} />
        <List title="Evidence" items={item.evidence} />
        <List title="Likely causes" items={item.likelyCauses} />
        <List title="Safe investigation" items={item.safeInvestigation} />
        <List title="Allowed remediation" items={item.allowedRemediation} />
        <List title="Verification" items={item.verification} />
      </div>
      <div className="mt-3">
        <Panel title="Escalation">
          <p className="text-sm text-muted-foreground">{item.escalation}</p>
        </Panel>
      </div>
    </div>
  );
}

function List({ title, items }: { title: string; items: string[] }) {
  return (
    <Panel title={title}>
      <ul className="list-disc space-y-1 pl-4 text-sm text-muted-foreground">
        {items.map((item) => (
          <li key={item}>{item}</li>
        ))}
      </ul>
    </Panel>
  );
}
