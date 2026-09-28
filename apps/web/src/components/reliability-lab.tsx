"use client";

import { useEffect, useState } from "react";
import { StatusBadge } from "@/components/status-badge";
import { ErrorBlock, LoadingBlock, PageHeader, Panel } from "@/components/states";
import { Button } from "@/components/ui/button";
import { Progress } from "@/components/ui/progress";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { apiPost } from "@/lib/api";
import { useApi } from "@/lib/use-api";
import type { Experiment, ExperimentCatalog, Service } from "@/lib/types";

const durations = [
  { value: "30", label: "30 seconds" },
  { value: "60", label: "60 seconds" },
  { value: "120", label: "2 minutes" },
];

export function ReliabilityLab() {
  const catalog = useApi<ExperimentCatalog>("/api/v1/experiments", 2000);
  const services = useApi<Service[]>("/api/v1/services");
  const [scenario, setScenario] = useState("database-connection-exhaustion");
  const [serviceId, setServiceId] = useState("payment-api");
  const [duration, setDuration] = useState("60");
  const [pending, setPending] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);
  const [now, setNow] = useState(() => Date.now());

  useEffect(() => {
    const id = setInterval(() => setNow(Date.now()), 500);
    return () => clearInterval(id);
  }, []);

  if (catalog.loading && !catalog.data) return <LoadingBlock />;
  if (!catalog.data) return <ErrorBlock message={catalog.error ?? "Lab unavailable"} onRetry={() => void catalog.reload()} />;

  const serviceItems = (services.data ?? []).map((service) => ({ value: service.id, label: service.name }));
  const selected = catalog.data.scenarios.find((item) => item.id === scenario);

  async function start() {
    setPending(true);
    setActionError(null);
    try {
      await apiPost<Experiment>("/api/v1/experiments", {
        serviceId,
        scenario,
        durationSec: Number(duration),
      });
      await catalog.reload();
    } catch (err) {
      setActionError(err instanceof Error ? err.message : "Experiment failed");
    } finally {
      setPending(false);
    }
  }

  return (
    <div>
      <PageHeader
        kicker="Simulation only"
        title="Reliability Lab"
        description={catalog.data.notice}
      />
      <div className="grid gap-3 lg:grid-cols-[minmax(0,1.4fr)_minmax(280px,0.8fr)]">
        <div className="grid gap-2 sm:grid-cols-2">
          {catalog.data.scenarios.map((item) => {
            const active = item.id === scenario;
            return (
              <button
                key={item.id}
                type="button"
                aria-pressed={active}
                onClick={() => setScenario(item.id)}
                className={`border px-3 py-2 text-left ${active ? "border-primary bg-muted" : "border-border bg-card hover:bg-muted/60"}`}
              >
                <div className="text-sm font-medium">{item.name}</div>
                <p className="mt-1 text-xs text-muted-foreground">{item.description}</p>
                <div className="mt-2 font-mono text-[11px] uppercase text-muted-foreground">{item.target}</div>
              </button>
            );
          })}
        </div>
        <Panel title="Run experiment">
          <div className="space-y-3">
            <label className="block text-xs text-muted-foreground">
              Service
              <Select
                items={serviceItems}
                value={serviceId}
                onValueChange={(value) => {
                  if (typeof value === "string") setServiceId(value);
                }}
              >
                <SelectTrigger className="mt-1 w-full">
                  <SelectValue placeholder="Select a service" />
                </SelectTrigger>
                <SelectContent>
                  {serviceItems.map((item) => (
                    <SelectItem key={item.value} value={item.value}>
                      {item.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </label>
            <label className="block text-xs text-muted-foreground">
              Failure scenario
              <div className="mt-1 text-sm text-foreground">{selected?.name ?? scenario}</div>
            </label>
            <label className="block text-xs text-muted-foreground">
              Duration
              <Select
                items={durations}
                value={duration}
                onValueChange={(value) => {
                  if (typeof value === "string") setDuration(value);
                }}
              >
                <SelectTrigger className="mt-1 w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {durations.map((item) => (
                    <SelectItem key={item.value} value={item.value}>
                      {item.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </label>
            <Button className="w-full" disabled={pending || !serviceId} onClick={() => void start()}>
              {pending ? "Starting…" : "Start simulated experiment"}
            </Button>
            {actionError ? <p className="text-xs text-red-300">{actionError}</p> : null}
            <p className="text-[11px] text-muted-foreground">
              The lab records the run and a timer. It does not change {serviceId} or open a new incident.
            </p>
          </div>
        </Panel>
      </div>
      <div className="mt-3">
        <Panel title="Runs" padded={false}>
          {catalog.data.runs.length === 0 ? (
            <p className="px-3 py-4 text-sm text-muted-foreground">No experiments yet.</p>
          ) : (
            <ul>
              {catalog.data.runs.map((run) => {
                const total = run.durationSec * 1000;
                const elapsed = Math.min(total, Math.max(0, now - Date.parse(run.startedAt)));
                const pct = total === 0 ? 100 : (elapsed / total) * 100;
                return (
                  <li key={run.id} className="border-b border-border px-3 py-3 last:border-b-0">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="font-mono text-sm">{run.id}</span>
                      <span className="text-sm font-medium">{run.scenarioName}</span>
                      <span className="text-sm text-muted-foreground">{run.serviceName}</span>
                      <StatusBadge value={run.status} />
                      <StatusBadge value="simulated" />
                    </div>
                    <Progress className="mt-2" value={run.status === "completed" ? 100 : pct} />
                    <p className="mt-1 text-[11px] text-muted-foreground">{run.note}</p>
                  </li>
                );
              })}
            </ul>
          )}
        </Panel>
      </div>
    </div>
  );
}
