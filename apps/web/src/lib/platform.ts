export type CheckVerdict = "PASS" | "WARNING" | "MISSING";

export interface ServiceContract {
  apiVersion: string;
  kind: string;
  metadata: { name: string };
  spec: {
    description: string;
    owner: string;
    team: string;
    runtime: string;
    language: string;
    repository: string;
    environment: string;
    namespace: string;
    version: string;
    health: string;
    slo: { availability: number; latencyMs: number; errorRate: number; window: string };
    dependencies: string[];
    runbook: string;
    dashboard: string;
    onCall: string;
    deploymentStrategy: string;
    lastDeployment: string;
    reliabilityStatus: string;
    checks: Record<string, CheckVerdict>;
    guidance?: Record<string, string>;
  };
}

export interface CatalogResponse {
  source: string;
  venue: string;
  services: ServiceContract[];
}

export interface Runbook {
  id: string;
  title: string;
  service: string;
  symptoms: string[];
  detection: string;
  evidence: string[];
  likelyCauses: string[];
  safeInvestigation: string[];
  allowedRemediation: string[];
  verification: string[];
  escalation: string[];
}

export interface ScorecardResponse {
  source: string;
  note: string;
  services: { name: string; checks: Record<string, CheckVerdict>; guidance?: Record<string, string>; source: string }[];
}

export const CHECK_LABELS: { key: string; label: string }[] = [
  { key: "sloDefined", label: "SLO defined" },
  { key: "runbook", label: "Runbook exists" },
  { key: "owner", label: "Owner defined" },
  { key: "healthEndpoint", label: "Health endpoint" },
  { key: "readinessProbe", label: "Readiness probe" },
  { key: "metrics", label: "Metrics" },
  { key: "tracing", label: "Tracing" },
  { key: "resourceLimits", label: "Resource limits" },
  { key: "ci", label: "CI" },
  { key: "gitops", label: "GitOps" },
  { key: "securityContext", label: "Security context" },
];
