"use client";

import Link from "next/link";
import { ErrorBlock, LoadingBlock, PageHeader, Panel } from "@/components/states";
import { CHECK_LABELS, type CatalogResponse, type ScorecardResponse } from "@/lib/platform";
import { isPortfolioRuntime } from "@/lib/runtime";
import { useApi } from "@/lib/use-api";

export function DeveloperPortal() {
  const catalog = useApi<CatalogResponse>("/api/v1/platform/catalog");
  const scorecard = useApi<ScorecardResponse>("/api/v1/platform/scorecard");
  const portfolio = isPortfolioRuntime();
  if (catalog.loading && !catalog.data) return <LoadingBlock label="Loading service catalog" />;
  if (!catalog.data) return <ErrorBlock message={catalog.error ?? "Catalog unavailable"} onRetry={() => void catalog.reload()} />;

  return (
    <div>
      <PageHeader
        kicker={portfolio ? "AWS — PORTFOLIO DEMO" : "LOCAL — LIVE"}
        title="Developer Portal"
        description="Service catalog, golden paths, ownership, SLOs, and runbooks for demo-shop. This is OpsPilot's internal platform view, not a Backstage replacement."
        actions={
          <Link href="/developer-portal/golden-path" className="inline-flex h-8 items-center rounded-[10px] border border-border bg-white px-3 text-sm font-medium hover:bg-secondary">
            Create service preview
          </Link>
        }
      />
      <div className="mb-3 grid gap-3 md:grid-cols-3">
        <Panel title="What is live">
          <p className="text-sm text-muted-foreground">
            {portfolio
              ? "These records are the service contract. Health numbers on the public site are sanitized samples, not Prometheus."
              : "demo-shop workloads on k3d are live. The contract below is the ownership and SLO record. Telemetry for payment-api is on the service page."}
          </p>
        </Panel>
        <Panel title="Golden path">
          <p className="text-sm text-muted-foreground">Go, Node.js, or Python. Rolling or canary. The preview includes a Deployment, probes, limits, and a non-root security context.</p>
        </Panel>
        <Panel title="Review order">
          <ol className="list-decimal space-y-1 pl-4 text-sm text-muted-foreground">
            <li>Overview</li>
            <li>This catalog</li>
            <li>SLOs and an incident</li>
            <li>Rollouts, architecture, security</li>
          </ol>
        </Panel>
      </div>
      <div className="grid gap-3 xl:grid-cols-[minmax(0,1.4fr)_minmax(280px,0.8fr)]">
        <Panel title="Service catalog" padded={false}>
          <ul>
            {catalog.data.services.map((service) => (
              <li key={service.metadata.name} className="border-t border-border first:border-t-0">
                <Link href={`/developer-portal/services/${service.metadata.name}`} className="block px-3 py-2 hover:bg-muted">
                  <div className="flex flex-wrap items-baseline gap-2">
                    <span className="font-medium">{service.metadata.name}</span>
                    <span className="font-mono text-[11px] text-muted-foreground">{service.spec.version}</span>
                    <span className="font-mono text-[11px] text-muted-foreground">{service.spec.runtime}</span>
                    <span className="ml-auto font-mono text-[11px] uppercase text-muted-foreground">{service.spec.deploymentStrategy}</span>
                  </div>
                  <p className="mt-1 text-sm text-muted-foreground">{service.spec.description}</p>
                  <p className="mt-1 font-mono text-[11px] text-muted-foreground">
                    {service.spec.owner} · {service.spec.namespace} · SLO {service.spec.slo.availability}% / {service.spec.slo.latencyMs}ms
                  </p>
                </Link>
              </li>
            ))}
          </ul>
        </Panel>
        <div className="space-y-3">
          <Panel title="Contract scorecard">
            <p className="mb-2 text-sm text-muted-foreground">{scorecard.data?.note ?? "Checks are PASS, WARNING, or MISSING. There is no invented score."}</p>
            <ul className="space-y-2">
              {(scorecard.data?.services ?? []).map((row) => {
                const missing = CHECK_LABELS.filter((item) => row.checks[item.key] === "MISSING").length;
                const warning = CHECK_LABELS.filter((item) => row.checks[item.key] === "WARNING").length;
                return (
                  <li key={row.name} className="flex items-baseline justify-between gap-2 text-sm">
                    <Link className="hover:underline" href={`/developer-portal/services/${row.name}`}>
                      {row.name}
                    </Link>
                    <span className="font-mono text-[11px] text-muted-foreground">
                      {missing} missing · {warning} warning
                    </span>
                  </li>
                );
              })}
            </ul>
          </Panel>
          <Panel title="Also in this console">
            <ul className="space-y-1 text-sm">
              <li><Link className="hover:underline" href="/runbooks">Runbooks</Link></li>
              <li><Link className="hover:underline" href="/slos">SLOs</Link></li>
              <li><Link className="hover:underline" href="/architecture">Architecture</Link></li>
              <li><Link className="hover:underline" href="/security">Security boundaries</Link></li>
            </ul>
          </Panel>
        </div>
      </div>
    </div>
  );
}
