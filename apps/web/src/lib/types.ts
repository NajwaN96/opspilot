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
  source: "simulation" | "kubernetes" | string;
  telemetry: "simulated" | "none" | string;
  image?: string;
  workloadKind?: string;
  restarts?: number;
  lastObserved?: string;
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
  likelyCause?: string;
  supporting?: string[];
  contradicting?: string[];
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
  audit?: AuditRecord[];
  origin?: string;
  metricsSource?: string;
  traceSource?: string;
  kubernetesSource?: string;
  ruleId?: string;
  telemetryRecoveredAt?: string;
  thresholds?: Record<string, string | number>;
}

export interface AuditRecord {
  id: number;
  at: string;
  actor: string;
  incidentId: string;
  proposalId?: string;
  action: string;
  policyResult: string;
  approvalResult: string;
  executionStatus: string;
  verificationResult: string;
  detail: string;
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
  realRuns?: Experiment[];
}

export interface Health {
  status: string;
  service: string;
  version: string;
}

export interface ReadyStatus {
  status: string;
  database: string;
  kubernetes: string;
  demoResetEnabled: boolean;
  prometheus?: string;
  opentelemetry?: string;
  traces?: string;
}

export interface TelemetrySnapshot {
  source: string;
  available: boolean;
  message?: string;
  service: string;
  namespace: string;
  updated?: string;
  requestRate: number;
  errorRate: number;
  requests: number;
  p50LatencyMs: number;
  p95LatencyMs: number;
  p99LatencyMs: number;
  availability: number;
  dataSource: string;
  slo?: {
    target: number;
    window: string;
    label: string;
    availability: number;
    errorBudgetConsumed: number;
    burnRate: number;
    sufficient: boolean;
    source: string;
  };
}

export interface TraceSummary {
  id: string;
  start: string;
  durationMs: number;
  rootService: string;
  status: string;
  spans: number;
}

export interface SpanView {
  service: string;
  operation: string;
  durationMs: number;
  status: string;
  depth: number;
  attributes?: Record<string, string>;
}

export interface TraceDetail {
  id: string;
  start: string;
  durationMs: number;
  rootService: string;
  status: string;
  spans: number;
  source: string;
  tree: SpanView[];
}

export interface TraceList {
  source: string;
  message?: string;
  traces: TraceSummary[];
}

export interface TelemetryStatus {
  prometheus: string;
  opentelemetry: string;
  traces: string;
}

export interface CloudPortfolio {
  accountState: "connected" | "not-connected" | string;
  accountMasked?: string;
  region?: string;
  authentication: string;
  cloudDeployment: string;
  kubernetesRuntime: string;
  paidCloudResources: string;
  intentionalInfrastructureSpend: string;
  eks: string;
}

export interface KubernetesStatus {
  cluster: string;
  mode: string;
  venue?: string;
  venueLabel?: string;
  namespace: string;
  namespaces: string[];
  connectivity: "connected" | "disconnected" | "degraded" | string;
  kubernetesVersion?: string;
  lastSync?: string;
  message?: string;
  source: string;
  nodeCount: number;
  nodesReady: number;
}

export interface WorkloadPod {
  name: string;
  namespace: string;
  service: string;
  status: string;
  ready: string;
  restarts: number;
  node: string;
  startedAt?: string;
  source: string;
}

export interface Workload {
  name: string;
  namespace: string;
  kind: string;
  version: string;
  image: string;
  desired: number;
  ready: number;
  restarts: number;
  status: string;
  lastObserved: string;
  serviceId: string;
  source: string;
  labels?: Record<string, string>;
  pods?: WorkloadPod[];
}

export interface ClusterEvent {
  id: string;
  at: string;
  namespace: string;
  type: string;
  reason: string;
  object: string;
  message: string;
  count: number;
  source: string;
}
