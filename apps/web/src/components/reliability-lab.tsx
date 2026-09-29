"use client";

import { useEffect, useState } from "react";
import { StatusBadge } from "@/components/status-badge";
import { ErrorBlock, LoadingBlock, PageHeader, Panel } from "@/components/states";
import { Button } from "@/components/ui/button";
import { Progress } from "@/components/ui/progress";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { apiPost } from "@/lib/api";
import { formatDuration, formatLatency, formatPercent } from "@/lib/format";
import { useApi } from "@/lib/use-api";
import type { Experiment, ExperimentCatalog, Service, TelemetrySnapshot } from "@/lib/types";

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
        kicker="Local cluster"
        title="Reliability Lab"
        description="Real experiments are limited to payment-api in demo-shop. The scenarios below that remain simulated do not change the cluster."
      />
      <CanaryExperiments />
      <RealExperiment />
      <BadRelease />
      <h2 className="mb-2 mt-6 text-sm font-medium uppercase tracking-wide text-muted-foreground">Simulated</h2>
      <p className="mb-3 text-sm text-muted-foreground">{catalog.data.notice}</p>
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

function CanaryExperiments() {
  const [pending, setPending] = useState<string | null>(null);
  const [message, setMessage] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);

  async function start(kind: "good" | "bad") {
    setPending(kind);
    setActionError(null);
    setMessage(null);
    try {
      const view = await apiPost<{ id: string; candidateVersion: string }>(`/api/v1/lab/payment-api/canary/${kind}`, {});
      setMessage(`${view.id} is deploying candidate ${view.candidateVersion}. Watch it on Rollouts. This is a local experiment.`);
    } catch (err) {
      setActionError(err instanceof Error ? err.message : "Canary failed");
    } finally {
      setPending(null);
    }
  }

  return (
    <Panel title="Canary experiments" className="mb-3">
      <p className="text-sm text-muted-foreground">
        These controls deploy a second payment-api beside stable 1.4.2 and shift 5%, then 25%, then 50% of checkout traffic. They do nothing when the API is in production mode. Promotion and abort still need a person.
      </p>
      <div className="mt-3 flex flex-wrap gap-2">
        <Button disabled={pending !== null} onClick={() => void start("good")}>
          {pending === "good" ? "Starting…" : "Start healthy canary 1.5.0"}
        </Button>
        <Button variant="outline" disabled={pending !== null} onClick={() => void start("bad")}>
          {pending === "bad" ? "Starting…" : "Start bad canary 1.6.0-bad"}
        </Button>
      </div>
      {message ? <p className="mt-3 text-sm">{message}</p> : null}
      {actionError ? <p className="mt-3 text-sm text-red-300">{actionError}</p> : null}
    </Panel>
  );
}

function BadRelease() {
  const [pending, setPending] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);
  const [message, setMessage] = useState<string | null>(null);

  async function deploy() {
    setPending(true);
    setActionError(null);
    setMessage(null);
    try {
      await apiPost<{ version: string }>("/api/v1/rollouts/payment-api/bad", {});
      setMessage("payment-api is ready on the known bad release 1.5.0-bad. This did not change the Reliability Lab fault.");
    } catch (err) {
      setActionError(err instanceof Error ? err.message : "Deploy failed");
    } finally {
      setPending(false);
    }
  }

  return (
    <div className="mt-3">
      <Panel title="Known bad release" action={<span className="text-[11px] uppercase tracking-wide text-amber-200">Development only</span>}>
        <p className="text-sm font-medium">Deploy Bad payment</p>
        <p className="mt-1 text-sm text-muted-foreground">
          Replaces demo-shop/payment-api with the server-known image for 1.5.0-bad. The pod stays Ready, adds 500ms, and returns HTTP 500 on about 30% of requests. The request does not accept an image name.
        </p>
        <div className="mt-3">
          <Button disabled={pending} onClick={() => void deploy()}>
            {pending ? "Deploying…" : "Deploy Bad payment"}
          </Button>
        </div>
        {message ? <p className="mt-2 text-sm text-amber-100">{message}</p> : null}
        {actionError ? <p className="mt-2 text-xs text-red-300">{actionError}</p> : null}
      </Panel>
    </div>
  );
}

function RealExperiment() {
  const catalog = useApi<ExperimentCatalog>("/api/v1/experiments", 2000);
  const telemetry = useApi<TelemetrySnapshot>("/api/v1/services/k8s_demo-shop_payment-api/telemetry", 2000);
  const [duration, setDuration] = useState("60");
  const [pending, setPending] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);
  const [now, setNow] = useState(() => Date.now());

  useEffect(() => {
    const id = setInterval(() => setNow(Date.now()), 500);
    return () => clearInterval(id);
  }, []);

  const running = catalog.data?.realRuns?.find((run) => run.status === "running");
  const elapsed = running ? Math.max(0, now - Date.parse(running.startedAt)) : 0;
  const remaining = running ? Math.max(0, Date.parse(running.endsAt) - now) : 0;

  async function start() {
    setPending(true);
    setActionError(null);
    try {
      await apiPost<Experiment>("/api/v1/experiments", {
        serviceId: "k8s_demo-shop_payment-api",
        scenario: "payment-api-degraded",
        durationSec: Number(duration),
      });
      await catalog.reload();
    } catch (err) {
      setActionError(err instanceof Error ? err.message : "Experiment failed");
    } finally {
      setPending(false);
    }
  }

  async function stop(id: string) {
    setPending(true);
    setActionError(null);
    try {
      await apiPost<Experiment>(`/api/v1/experiments/${id}/stop`, {});
      await catalog.reload();
    } catch (err) {
      setActionError(err instanceof Error ? err.message : "Stop failed");
    } finally {
      setPending(false);
    }
  }

  return (
    <Panel title="Real local experiment" action={<span className="text-[11px] uppercase tracking-wide text-amber-200">Real local experiment</span>}>
      <p className="text-sm font-medium text-amber-100">Local demo-shop only. Experiment automatically expires.</p>
      <p className="mt-1 text-sm text-muted-foreground">
        payment-api answers 40% of requests with HTTP 500 and adds 500ms of latency. The fault lives in the process and does not change the Deployment.
      </p>
      <div className="mt-3 grid gap-3 sm:grid-cols-3">
        <div>
          <div className="text-[11px] uppercase tracking-wide text-muted-foreground">Service</div>
          <div className="mt-1 font-mono text-sm">payment-api</div>
        </div>
        <div>
          <div className="text-[11px] uppercase tracking-wide text-muted-foreground">Scenario</div>
          <div className="mt-1 text-sm">Elevated Latency + Errors</div>
        </div>
        <label className="block text-xs text-muted-foreground">
          Duration
          <Select items={durations} value={duration} onValueChange={(value) => { if (typeof value === "string") setDuration(value); }}>
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
      </div>
      <div className="mt-3 flex flex-wrap gap-2">
        <Button disabled={pending || Boolean(running)} onClick={() => void start()}>
          {pending && !running ? "Starting…" : "Start Experiment"}
        </Button>
        {running ? (
          <Button variant="outline" disabled={pending} onClick={() => void stop(running.id)}>
            Stop Experiment
          </Button>
        ) : null}
      </div>
      {actionError ? <p className="mt-2 text-xs text-red-300">{actionError}</p> : null}
      {running ? (
        <div className="mt-3 grid gap-3 border border-amber-400/30 bg-amber-400/10 p-3 sm:grid-cols-2 lg:grid-cols-4">
          <Stat label="Status" value="Experiment Running" />
          <Stat label="Elapsed" value={formatDuration(elapsed)} />
          <Stat label="Remaining" value={formatDuration(remaining)} />
          <Stat label="Injected conditions" value="40% HTTP 500 · +500ms" />
          <Stat label="Observed error rate" value={telemetry.data?.available ? formatPercent(telemetry.data.errorRate * 100) : "Telemetry unavailable"} />
          <Stat label="Observed p95" value={telemetry.data?.available ? formatLatency(telemetry.data.p95LatencyMs) : "Telemetry unavailable"} />
        </div>
      ) : null}
      {(catalog.data?.realRuns ?? []).length > 0 ? (
        <ul className="mt-3">
          {catalog.data?.realRuns?.map((run) => (
            <li key={run.id} className="flex flex-wrap items-center gap-2 border-t border-border py-2 text-sm">
              <span className="font-mono text-xs">{run.id}</span>
              <StatusBadge value={run.status} />
              <span className="text-xs uppercase tracking-wide text-amber-200">Real local experiment</span>
              <span className="text-xs text-muted-foreground">{run.note}</span>
            </li>
          ))}
        </ul>
      ) : null}
    </Panel>
  );
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <div className="text-[11px] uppercase tracking-wide text-muted-foreground">{label}</div>
      <div className="mt-1 font-mono text-sm">{value}</div>
    </div>
  );
}
