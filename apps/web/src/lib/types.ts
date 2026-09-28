export type ServiceStatus = "healthy" | "degraded" | "critical" | string;
export type IncidentStatus = "investigating" | "mitigating" | "resolved" | string;

export interface ClusterHealth {
  state: string;
  activeIncidents: number;
  sloCompliance: number;
  servicesWithinSLO: number;
  servicesTotal: number;
  servicesHealthy: number;
  deploymentsToday: number;
}

export interface Node {
  name: string;
  status: string;
  role: string;
  cpuPercent: number;
  memoryPercent: number;
  pods: number;
  podCapacity: number;
  zone: string;
  kubeletVersion: string;
}

export interface Namespace {
  name: string;
  services: number;
  status: string;
}

export interface ComponentHealth {
  name: string;
  status: string;
}

export interface Cluster {
  id: string;
  name: string;
  environment: string;
  region: string;
  kubernetesVersion: string;
  status: string;
  provider: string;
  simulated: boolean;
  health: ClusterHealth;
  nodes: Node[];
  namespaces: Namespace[];
  controlPlane: ComponentHealth[];
}

export interface SLO {
  objective: number;
  window: string;
  compliance: number;
  errorBudgetRemaining: number;
  burnRate: number;
  withinSLO: boolean;
}

export interface Replicas {
  desired: number;
  ready: number;
}

export interface Dependency {
  id: string;
  name: string;
  kind: string;
  relation: string;
  status: string;
}

export interface Deployment {
  id: string;
  serviceId: string;
  serviceName: string;
  version: string;
  previousVersion: string;
  at: string;
  clock: string;
  actor: string;
  status: string;
  strategy: string;
  change: string;
}

export interface MetricPoint {
  clock: string;
  p95LatencyMs: number;
  errorRate: number;
  dbConnections: number;
  requestRate: number;
}

export interface Service {
  id: string;
  name: string;
  clusterId: string;
  namespace: string;
  status: ServiceStatus;
  owner: string;
  runtime: string;
  description: string;
  version: string;
  previousVersion: string;
  availability: number;
  p95LatencyMs: number;
  errorRate: number;
  slo: SLO;
  lastDeployment: string;
  replicas: Replicas;
  dependencies?: Dependency[];
  deployments?: Deployment[];
  metrics?: MetricPoint[];
  openIncidents?: string[];
}

export interface Snapshot {
  availability: number;
  p95LatencyMs: number;
  errorRate: number;
  dbConnections: number;
  status: string;
  version: string;
}

export interface TimelineEvent {
  clock: string;
  at: string;
  title: string;
  detail: string;
  kind: string;
}

export interface LogLine {
  clock: string;
  level: string;
  service: string;
  message: string;
}

export interface Span {
  id: string;
  parentId?: string;
  service: string;
  name: string;
  durationMs: number;
  status: string;
  slow: boolean;
}

export interface Trace {
  id: string;
  traceId: string;
  clock: string;
  spans: Span[];
}

export interface KubeEvent {
  clock: string;
  type: string;
  reason: string;
  object: string;
  message: string;
}

export interface Evidence {
  metrics: MetricPoint[];
  logs: LogLine[];
  traces: Trace[];
  kubernetesEvents: KubeEvent[];
  deployments: Deployment[];
}

export interface Analysis {
  simulated: boolean;
  cause: string;
  confidence: number;
  evidence: string[];
}

export interface Recommendation {
  action: string;
  summary: string;
  from: string;
  to: string;
  risk: string;
  policy: string;
  allowed: boolean;
}

export interface RemediationStep {
  name: string;
  status: "pending" | "active" | "complete" | string;
  at?: string;
  clock?: string;
}

export interface Remediation {
  id: string;
  action: string;
  from: string;
  to: string;
  status: string;
  simulated: boolean;
  startedAt: string;
  steps: RemediationStep[];
}

export interface Incident {
  id: string;
  severity: string;
  serviceId: string;
  serviceName: string;
  title: string;
  status: IncidentStatus;
  startedAt: string;
  resolvedAt?: string;
  durationSec: number;
  summary: string;
  cluster: string;
  timeline?: TimelineEvent[];
  evidence?: Evidence;
  analysis?: Analysis;
  recommendation?: Recommendation;
  remediation?: Remediation;
  snapshot: Snapshot;
}

export interface Scenario {
  id: string;
  name: string;
  description: string;
  target: string;
}

export interface Experiment {
  id: string;
  serviceId: string;
  serviceName: string;
  scenario: string;
  scenarioName: string;
  durationSec: number;
  status: string;
  simulated: boolean;
  startedAt: string;
  endsAt: string;
  note: string;
}

export interface ExperimentCatalog {
  simulated: boolean;
  notice: string;
  scenarios: Scenario[];
  runs: Experiment[];
}

export interface Health {
  status: string;
  service: string;
  version: string;
}
