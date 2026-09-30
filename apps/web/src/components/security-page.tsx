"use client";

import { PageHeader, Panel } from "@/components/states";
import { isPortfolioRuntime } from "@/lib/runtime";

const rows = [
  ["AI", "READ ONLY", "Evidence snapshot in, structured recommendation out. No kubectl, shell, SQL, or PromQL from the model."],
  ["Public AWS", "READ ONLY", "GET serves the console and sample API. POST returns 403."],
  ["Kubernetes discovery", "READ ONLY", "The local client lists workloads. It is not a general admin client."],
  ["Remediation", "ALLOWLISTED", "One payment-api rollback, from opspilot-demo:1.5.0-bad to opspilot-demo:1.4.2, on demo-shop."],
  ["Executor", "CONSTRAINED", "A person approves. The executor does not choose a different image or namespace."],
  ["AWS deployment", "OIDC + LEAST PRIVILEGE", "GitHub assumes opspilot-github-deployer. The role updates the existing Lambda and reads the free-plan state."],
  ["EKS", "PLAN ONLY", "OpenTofu validate only. No cluster, nodes, NAT, or load balancer are created."],
  ["Secrets", "NOT STORED IN GIT", "No access keys, kubeconfig, tfstate, or API tokens belong in the repository."],
];

export function SecurityPage() {
  const portfolio = isPortfolioRuntime();
  return (
    <div>
      <PageHeader
        kicker={portfolio ? "AWS — PORTFOLIO DEMO" : "LOCAL — LIVE"}
        title="Security"
        description="Human approval sits between a recommendation and a cluster change. The model and the public site are not on that path."
      />
      <Panel title="Boundaries" padded={false}>
        <ul>
          {rows.map(([name, mode, detail]) => (
            <li key={name} className="grid gap-1 border-t border-border px-3 py-2 first:border-t-0 md:grid-cols-[10rem_14rem_minmax(0,1fr)]">
              <div className="text-sm font-medium">{name}</div>
              <div className="font-mono text-[11px] text-status-info">{mode}</div>
              <div className="text-sm text-muted-foreground">{detail}</div>
            </li>
          ))}
        </ul>
      </Panel>
      <div className="mt-3 grid gap-3 md:grid-cols-2">
        <Panel title="Cost policy">
          <dl className="space-y-2 text-sm">
            <Row label="AWS account mode" value="FREE" />
            <Row label="Public portfolio" value="Lambda" />
            <Row label="EKS" value="PLAN ONLY" />
            <Row label="Paid cloud override" value="DISABLED" />
          </dl>
          <p className="mt-2 text-sm text-muted-foreground">This is the project policy. It is not an AWS invoice. Cost Explorer is not queried.</p>
        </Panel>
        <Panel title="Approval">
          <p className="text-sm text-muted-foreground">
            Detection and diagnosis can run without a person. Changing payment-api cannot. The public portfolio does not host the executor, so the same button there is refused.
          </p>
        </Panel>
      </div>
    </div>
  );
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex justify-between gap-3">
      <dt className="text-muted-foreground">{label}</dt>
      <dd className="font-mono text-xs">{value}</dd>
    </div>
  );
}
