"use client";

import { PageHeader, Panel } from "@/components/states";

export function ArchitecturePage() {
  return (
    <div>
      <PageHeader
        kicker="Three venues"
        title="Architecture"
        description="The public site, the local lab, and the plan-only EKS stack are different systems. They are not one cluster with three names."
      />
      <div className="grid gap-3 lg:grid-cols-3">
        <Panel title="Public AWS portfolio">
          <pre className="overflow-auto text-xs text-muted-foreground">{`GitHub
  → GitHub Actions
  → OIDC
  → AWS STS
  → opspilot-github-deployer
  → Lambda opspilot-portfolio
  → public function URL`}</pre>
          <p className="mt-2 text-sm text-muted-foreground">Read-only console and sample API. POST is refused. No access keys.</p>
        </Panel>
        <Panel title="Local live lab">
          <pre className="overflow-auto text-xs text-muted-foreground">{`Developer
  → Git
  → CI
  → GitOps manifests
  → k3d opspilot-dev
  → demo-shop
  → Prometheus / OpenTelemetry / Jaeger
  → detection
  → incident
  → investigation
  → approval
  → constrained executor
  → verification`}</pre>
          <p className="mt-2 text-sm text-muted-foreground">payment-api telemetry and the rollback executor run here.</p>
        </Panel>
        <Panel title="Plan-only cloud">
          <pre className="overflow-auto text-xs text-muted-foreground">{`OpenTofu
  → VPC
  → EKS
  → node group
  → ECR

PLAN ONLY — NOT DEPLOYED`}</pre>
          <p className="mt-2 text-sm text-muted-foreground">CI validates the stack. It does not apply it. The paid override stays unset.</p>
        </Panel>
      </div>
      <div className="mt-3">
        <Panel title="Platform path">
          <pre className="overflow-auto text-xs text-muted-foreground">{`Developer portal
  → service catalog
  → golden path / template
  → SLO and ownership
  → runbook
  → Git
  → CI
  → GitOps
  → Kubernetes`}</pre>
        </Panel>
      </div>
    </div>
  );
}
