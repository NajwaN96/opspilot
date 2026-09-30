"use client";

import Link from "next/link";
import { ErrorBlock, LoadingBlock, PageHeader, Panel } from "@/components/states";
import { CHECK_LABELS, type ServiceContract } from "@/lib/platform";
import { isPortfolioRuntime } from "@/lib/runtime";
import { useApi } from "@/lib/use-api";

function verdictClass(value: string) {
  if (value === "PASS") return "text-status-success";
  if (value === "WARNING") return "text-status-warning";
  return "text-status-danger";
}

export function ContractDetail({ name }: { name: string }) {
  const service = useApi<ServiceContract>(`/api/v1/platform/catalog/${name}`);
  const portfolio = isPortfolioRuntime();
  if (service.loading && !service.data) return <LoadingBlock label="Loading service contract" />;
  if (!service.data?.spec) return <ErrorBlock message={service.error ?? "Service is not in the catalog"} onRetry={() => void service.reload()} />;
  const spec = service.data.spec;
  const liveId = `k8s_demo-shop_${name}`;
  return (
    <div>
      <PageHeader
        kicker={portfolio ? "Contract · AWS demo does not scrape the cluster" : "Contract · local workloads are separate records"}
        title={name}
        description={spec.description}
      />
      <dl className="mb-3 grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <Fact label="Owner" value={spec.owner} />
        <Fact label="Runtime" value={spec.runtime} />
        <Fact label="Version" value={spec.version} mono />
        <Fact label="Namespace" value={spec.namespace} mono />
        <Fact label="Strategy" value={spec.deploymentStrategy} />
        <Fact label="On call" value={spec.onCall} />
        <Fact label="Availability SLO" value={`${spec.slo.availability}% / ${spec.slo.window}`} mono />
        <Fact label="Latency SLO" value={`${spec.slo.latencyMs}ms p95`} mono />
      </dl>
      <div className="grid gap-3 lg:grid-cols-2">
        <Panel title="Delivery">
          <p className="text-sm text-muted-foreground">Repository: {spec.repository}</p>
          <p className="mt-2 text-sm text-muted-foreground">Last recorded delivery: {spec.lastDeployment}</p>
          <p className="mt-2 text-sm text-muted-foreground">Reliability: {spec.reliabilityStatus}</p>
          <p className="mt-2 text-sm">
            <Link className="hover:underline" href={`/services/${liveId}`}>Kubernetes service view</Link>
            {" · "}
            <Link className="hover:underline" href={`/runbooks/${spec.runbook}`}>Runbook</Link>
          </p>
        </Panel>
        <Panel title="Dependencies">
          {spec.dependencies.length === 0 ? (
            <p className="text-sm text-muted-foreground">No upstream services in the contract.</p>
          ) : (
            <ul className="space-y-1 text-sm">
              {spec.dependencies.map((item) => (
                <li key={item}>
                  <Link className="hover:underline" href={`/developer-portal/services/${item}`}>{item}</Link>
                </li>
              ))}
            </ul>
          )}
          <p className="mt-3 text-sm text-muted-foreground">{spec.dashboard}</p>
        </Panel>
        <Panel title="SLO">
          <p className="text-sm text-muted-foreground">
            Target availability {spec.slo.availability}%. Latency {spec.slo.latencyMs}ms. Error rate under {spec.slo.errorRate}%. Window {spec.slo.window}.
          </p>
          <p className="mt-2 text-sm text-muted-foreground">
            {portfolio
              ? "The public console does not compute a live error budget. Sample budgets on the SLO page are labeled portfolio data."
              : "payment-api error rate and p95 on the local service page come from Prometheus. Other services keep the contract target until they have a detector."}
          </p>
        </Panel>
        <Panel title="Scorecard" padded={false}>
          <ul>
            {CHECK_LABELS.map((item) => (
              <li key={item.key} className="flex items-start justify-between gap-3 border-t border-border px-3 py-1.5 first:border-t-0">
                <span className="text-sm">{item.label}</span>
                <span className={`font-mono text-[11px] ${verdictClass(spec.checks[item.key] ?? "MISSING")}`}>{spec.checks[item.key] ?? "MISSING"}</span>
              </li>
            ))}
          </ul>
          {spec.guidance ? (
            <ul className="border-t border-border px-3 py-2 text-sm text-muted-foreground">
              {Object.entries(spec.guidance).map(([key, text]) => (
                <li key={key}>{text}</li>
              ))}
            </ul>
          ) : null}
        </Panel>
      </div>
    </div>
  );
}

function Fact({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div className="surface-card overflow-hidden rounded-xl px-3 py-2">
      <dt className="text-[11px] uppercase tracking-wide text-muted-foreground">{label}</dt>
      <dd className={`mt-1 text-sm ${mono ? "font-mono" : ""}`}>{value}</dd>
    </div>
  );
}
