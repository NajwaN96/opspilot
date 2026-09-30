"use client";

import { useState } from "react";
import { PageHeader, Panel } from "@/components/states";
import { Button } from "@/components/ui/button";
import { apiPost } from "@/lib/api";
import { previewError, previewFiles, type GoldenRuntime, type GoldenStrategy } from "@/lib/golden-path";
import { isPortfolioRuntime } from "@/lib/runtime";

export function GoldenPathPage() {
  const portfolio = isPortfolioRuntime();
  const [name, setName] = useState("billing-api");
  const [runtime, setRuntime] = useState<GoldenRuntime>("go");
  const [owner, setOwner] = useState("payments");
  const [slo, setSlo] = useState("99.9");
  const [strategy, setStrategy] = useState<GoldenStrategy>("canary");
  const [writeMessage, setWriteMessage] = useState<string | null>(null);
  const [pending, setPending] = useState(false);
  const input = { name, runtime, owner, slo, strategy };
  const error = previewError(input);
  const files = error ? [] : previewFiles(input);

  async function writeLocal() {
    setPending(true);
    setWriteMessage(null);
    try {
      const result = await apiPost<{ path: string }>("/api/v1/platform/golden-path", input);
      setWriteMessage(`Wrote ${result.path}. Existing demo-shop services are refused.`);
    } catch (err) {
      setWriteMessage(err instanceof Error ? err.message : "Write refused");
    } finally {
      setPending(false);
    }
  }

  return (
    <div>
      <PageHeader
        kicker={portfolio ? "PREVIEW / DEMO ONLY" : "LOCAL — LIVE"}
        title="Golden path"
        description="Choose a name, runtime, owner, SLO, and deployment strategy. The preview is a skeleton. It does not create a Git repository or a cluster."
      />
      <div className="grid gap-3 lg:grid-cols-[280px_minmax(0,1fr)]">
        <Panel title="Create new service">
          <label className="mb-2 block text-sm">
            Service name
            <input className="mt-1 w-full border border-border bg-background px-2 py-1 font-mono text-sm" value={name} onChange={(event) => setName(event.target.value)} />
          </label>
          <label className="mb-2 block text-sm">
            Runtime
            <select className="mt-1 w-full border border-border bg-background px-2 py-1 text-sm" value={runtime} onChange={(event) => setRuntime(event.target.value as GoldenRuntime)}>
              <option value="go">Go</option>
              <option value="node">Node.js</option>
              <option value="python">Python</option>
            </select>
          </label>
          <label className="mb-2 block text-sm">
            Team / owner
            <input className="mt-1 w-full border border-border bg-background px-2 py-1 text-sm" value={owner} onChange={(event) => setOwner(event.target.value)} />
          </label>
          <label className="mb-2 block text-sm">
            Availability SLO
            <input className="mt-1 w-full border border-border bg-background px-2 py-1 font-mono text-sm" value={slo} onChange={(event) => setSlo(event.target.value)} />
          </label>
          <label className="mb-3 block text-sm">
            Deployment strategy
            <select className="mt-1 w-full border border-border bg-background px-2 py-1 text-sm" value={strategy} onChange={(event) => setStrategy(event.target.value as GoldenStrategy)}>
              <option value="rolling">Rolling</option>
              <option value="canary">Canary</option>
            </select>
          </label>
          {error ? <p className="text-sm text-status-warning">{error}</p> : null}
          {portfolio ? (
            <p className="text-sm text-muted-foreground">The public site cannot write files. Use the local lab to copy a template into generated/services.</p>
          ) : (
            <Button type="button" disabled={Boolean(error) || pending} onClick={() => void writeLocal()}>
              {pending ? "Writing" : "Write local preview"}
            </Button>
          )}
          {writeMessage ? <p className="mt-2 text-sm text-muted-foreground">{writeMessage}</p> : null}
        </Panel>
        <Panel title="Preview" padded={false}>
          {files.length === 0 ? (
            <p className="px-3 py-2 text-sm text-muted-foreground">Fix the form to see the skeleton.</p>
          ) : (
            <ul>
              {files.map((file) => (
                <li key={file.path} className="border-t border-border first:border-t-0">
                  <div className="px-3 py-1 font-mono text-[11px] text-muted-foreground">{file.path}</div>
                  <pre className="overflow-auto px-3 pb-3 text-xs">{file.body}</pre>
                </li>
              ))}
            </ul>
          )}
        </Panel>
      </div>
    </div>
  );
}
