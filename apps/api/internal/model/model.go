package model

import "time"

// Status values shared across services and clusters.
const (
	StatusHealthy  = "healthy"
	StatusDegraded = "degraded"
	StatusCritical = "critical"
	StatusIdle     = "idle"
)

// Incident workflow states.
const (
	IncidentInvestigating = "investigating"
	IncidentMitigating    = "mitigating"
	IncidentResolved      = "resolved"
)

type Cluster struct {
	ID                string        `json:"id"`
	Name              string        `json:"name"`
	Environment       string        `json:"environment"`
	Region            string        `json:"region"`
	KubernetesVersion string        `json:"kubernetesVersion"`
	Status            string        `json:"status"`
	Provider          string        `json:"provider"`
	Simulated         bool          `json:"simulated"`
	Health            ClusterHealth `json:"health"`
	Nodes             []Node        `json:"nodes"`
	Namespaces        []Namespace   `json:"namespaces"`
	ControlPlane      []Component   `json:"controlPlane"`
}

type ClusterHealth struct {
	State             string  `json:"state"`
	ActiveIncidents   int     `json:"activeIncidents"`
	SLOCompliance     float64 `json:"sloCompliance"`
	ServicesWithinSLO int     `json:"servicesWithinSLO"`
	ServicesTotal     int     `json:"servicesTotal"`
	ServicesHealthy   int     `json:"servicesHealthy"`
	DeploymentsToday  int     `json:"deploymentsToday"`
}

type Node struct {
	Name           string  `json:"name"`
	Status         string  `json:"status"`
	Role           string  `json:"role"`
	CPUPercent     float64 `json:"cpuPercent"`
	MemoryPercent  float64 `json:"memoryPercent"`
	Pods           int     `json:"pods"`
	PodCapacity    int     `json:"podCapacity"`
	Zone           string  `json:"zone"`
	KubeletVersion string  `json:"kubeletVersion"`
}

type Namespace struct {
	Name     string `json:"name"`
	Services int    `json:"services"`
	Status   string `json:"status"`
}

type Component struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type Service struct {
	ID              string        `json:"id"`
	Name            string        `json:"name"`
	ClusterID       string        `json:"clusterId"`
	Namespace       string        `json:"namespace"`
	Status          string        `json:"status"`
	Owner           string        `json:"owner"`
	Runtime         string        `json:"runtime"`
	Description     string        `json:"description"`
	Version         string        `json:"version"`
	PreviousVersion string        `json:"previousVersion"`
	Availability    float64       `json:"availability"`
	P95LatencyMs    int           `json:"p95LatencyMs"`
	ErrorRate       float64       `json:"errorRate"`
	SLO             SLO           `json:"slo"`
	LastDeployment  time.Time     `json:"lastDeployment"`
	Replicas        Replicas      `json:"replicas"`
	Dependencies    []Dependency  `json:"dependencies,omitempty"`
	Deployments     []Deployment  `json:"deployments,omitempty"`
	Metrics         []MetricPoint `json:"metrics,omitempty"`
	OpenIncidents   []string      `json:"openIncidents,omitempty"`
	// Source is "simulation" or "kubernetes". Telemetry is "simulated" or "none".
	Source       string     `json:"source"`
	Telemetry    string     `json:"telemetry"`
	Image        string     `json:"image,omitempty"`
	WorkloadKind string     `json:"workloadKind,omitempty"`
	Restarts     int        `json:"restarts,omitempty"`
	LastObserved *time.Time `json:"lastObserved,omitempty"`
}

type SLO struct {
	Objective            float64 `json:"objective"`
	Window               string  `json:"window"`
	Compliance           float64 `json:"compliance"`
	ErrorBudgetRemaining float64 `json:"errorBudgetRemaining"`
	BurnRate             float64 `json:"burnRate"`
	WithinSLO            bool    `json:"withinSLO"`
}

type Replicas struct {
	Desired int `json:"desired"`
	Ready   int `json:"ready"`
}

type Dependency struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	Relation string `json:"relation"`
	Status   string `json:"status"`
}

type Deployment struct {
	ID              string    `json:"id"`
	ServiceID       string    `json:"serviceId"`
	ServiceName     string    `json:"serviceName"`
	Version         string    `json:"version"`
	PreviousVersion string    `json:"previousVersion"`
	At              time.Time `json:"at"`
	Clock           string    `json:"clock"`
	Actor           string    `json:"actor"`
	Status          string    `json:"status"`
	Strategy        string    `json:"strategy"`
	Change          string    `json:"change"`
}

type MetricPoint struct {
	Clock         string  `json:"clock"`
	P95LatencyMs  int     `json:"p95LatencyMs"`
	ErrorRate     float64 `json:"errorRate"`
	DBConnections float64 `json:"dbConnections"`
	RequestRate   float64 `json:"requestRate"`
}

type Incident struct {
	ID                   string          `json:"id"`
	Severity             string          `json:"severity"`
	ServiceID            string          `json:"serviceId"`
	ServiceName          string          `json:"serviceName"`
	Title                string          `json:"title"`
	Status               string          `json:"status"`
	StartedAt            time.Time       `json:"startedAt"`
	ResolvedAt           *time.Time      `json:"resolvedAt,omitempty"`
	DurationSec          int             `json:"durationSec"`
	Summary              string          `json:"summary"`
	Cluster              string          `json:"cluster"`
	Timeline             []TimelineEvent `json:"timeline,omitempty"`
	Evidence             *Evidence       `json:"evidence,omitempty"`
	Analysis             *Analysis       `json:"analysis,omitempty"`
	Recommendation       *Recommendation `json:"recommendation,omitempty"`
	Remediation          *Remediation    `json:"remediation,omitempty"`
	Snapshot             Snapshot        `json:"snapshot"`
	Audit                []AuditRecord   `json:"audit,omitempty"`
	Origin               string          `json:"origin,omitempty"`
	MetricsSource        string          `json:"metricsSource,omitempty"`
	TraceSource          string          `json:"traceSource,omitempty"`
	KubernetesSource     string          `json:"kubernetesSource,omitempty"`
	RuleID               string          `json:"ruleId,omitempty"`
	TelemetryRecoveredAt *time.Time      `json:"telemetryRecoveredAt,omitempty"`
	Thresholds           map[string]any  `json:"thresholds,omitempty"`
}

// AuditRecord is an append-only remediation action. It is not a Kubernetes event.
type AuditRecord struct {
	ID                 int64     `json:"id"`
	At                 time.Time `json:"at"`
	Actor              string    `json:"actor"`
	IncidentID         string    `json:"incidentId"`
	ProposalID         string    `json:"proposalId,omitempty"`
	Action             string    `json:"action"`
	PolicyResult       string    `json:"policyResult"`
	ApprovalResult     string    `json:"approvalResult"`
	ExecutionStatus    string    `json:"executionStatus"`
	VerificationResult string    `json:"verificationResult"`
	Detail             string    `json:"detail"`
}

type Snapshot struct {
	Availability  float64 `json:"availability"`
	P95LatencyMs  int     `json:"p95LatencyMs"`
	ErrorRate     float64 `json:"errorRate"`
	DBConnections float64 `json:"dbConnections"`
	Status        string  `json:"status"`
	Version       string  `json:"version"`
}

type TimelineEvent struct {
	Clock  string    `json:"clock"`
	At     time.Time `json:"at"`
	Title  string    `json:"title"`
	Detail string    `json:"detail"`
	Kind   string    `json:"kind"`
}

type Evidence struct {
	Metrics          []MetricPoint `json:"metrics"`
	Logs             []LogLine     `json:"logs"`
	Traces           []Trace       `json:"traces"`
	KubernetesEvents []KubeEvent   `json:"kubernetesEvents"`
	Deployments      []Deployment  `json:"deployments"`
}

type LogLine struct {
	Clock   string `json:"clock"`
	Level   string `json:"level"`
	Service string `json:"service"`
	Message string `json:"message"`
}

type Trace struct {
	ID      string `json:"id"`
	TraceID string `json:"traceId"`
	Clock   string `json:"clock"`
	Spans   []Span `json:"spans"`
}

type Span struct {
	ID         string `json:"id"`
	ParentID   string `json:"parentId,omitempty"`
	Service    string `json:"service"`
	Name       string `json:"name"`
	DurationMs int    `json:"durationMs"`
	Status     string `json:"status"`
	Slow       bool   `json:"slow"`
}

type KubeEvent struct {
	Clock   string `json:"clock"`
	Type    string `json:"type"`
	Reason  string `json:"reason"`
	Object  string `json:"object"`
	Message string `json:"message"`
}

type Analysis struct {
	Simulated     bool     `json:"simulated"`
	Cause         string   `json:"cause"`
	Confidence    float64  `json:"confidence"`
	Evidence      []string `json:"evidence"`
	LikelyCause   string   `json:"likelyCause,omitempty"`
	Supporting    []string `json:"supporting,omitempty"`
	Contradicting []string `json:"contradicting,omitempty"`
}

type Recommendation struct {
	Action  string `json:"action"`
	Summary string `json:"summary"`
	From    string `json:"from"`
	To      string `json:"to"`
	Risk    string `json:"risk"`
	Policy  string `json:"policy"`
	Allowed bool   `json:"allowed"`
}

type Remediation struct {
	ID        string            `json:"id"`
	Action    string            `json:"action"`
	From      string            `json:"from"`
	To        string            `json:"to"`
	Status    string            `json:"status"`
	Simulated bool              `json:"simulated"`
	StartedAt time.Time         `json:"startedAt"`
	Steps     []RemediationStep `json:"steps"`
}

type RemediationStep struct {
	Name   string     `json:"name"`
	Status string     `json:"status"`
	At     *time.Time `json:"at,omitempty"`
	Clock  string     `json:"clock,omitempty"`
}

type Scenario struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Target      string `json:"target"`
}

type Experiment struct {
	ID           string    `json:"id"`
	ServiceID    string    `json:"serviceId"`
	ServiceName  string    `json:"serviceName"`
	Scenario     string    `json:"scenario"`
	ScenarioName string    `json:"scenarioName"`
	DurationSec  int       `json:"durationSec"`
	Status       string    `json:"status"`
	Simulated    bool      `json:"simulated"`
	StartedAt    time.Time `json:"startedAt"`
	EndsAt       time.Time `json:"endsAt"`
	Note         string    `json:"note"`
}

type ExperimentCatalog struct {
	Simulated bool         `json:"simulated"`
	Notice    string       `json:"notice"`
	Scenarios []Scenario   `json:"scenarios"`
	Runs      []Experiment `json:"runs"`
	RealRuns  []Experiment `json:"realRuns,omitempty"`
}

// KubernetesStatus is the control plane's view of a real cluster.
// It is never used for the simulated production-01 incident plane.
type KubernetesStatus struct {
	Cluster           string     `json:"cluster"`
	Mode              string     `json:"mode"`
	Namespace         string     `json:"namespace"`
	Namespaces        []string   `json:"namespaces"`
	Connectivity      string     `json:"connectivity"`
	KubernetesVersion string     `json:"kubernetesVersion,omitempty"`
	LastSync          *time.Time `json:"lastSync,omitempty"`
	Message           string     `json:"message,omitempty"`
	Source            string     `json:"source"`
	NodeCount         int        `json:"nodeCount"`
	NodesReady        int        `json:"nodesReady"`
}

type Workload struct {
	Name         string            `json:"name"`
	Namespace    string            `json:"namespace"`
	Kind         string            `json:"kind"`
	Version      string            `json:"version"`
	Image        string            `json:"image"`
	Desired      int               `json:"desired"`
	Ready        int               `json:"ready"`
	Restarts     int               `json:"restarts"`
	Status       string            `json:"status"`
	LastObserved time.Time         `json:"lastObserved"`
	ServiceID    string            `json:"serviceId"`
	Source       string            `json:"source"`
	Labels       map[string]string `json:"labels,omitempty"`
	Pods         []Pod             `json:"pods,omitempty"`
}

type Pod struct {
	Name      string     `json:"name"`
	Namespace string     `json:"namespace"`
	Service   string     `json:"service"`
	Status    string     `json:"status"`
	Ready     string     `json:"ready"`
	Restarts  int        `json:"restarts"`
	Node      string     `json:"node"`
	StartedAt *time.Time `json:"startedAt,omitempty"`
	Source    string     `json:"source"`
}

type ClusterEvent struct {
	ID        string    `json:"id"`
	At        time.Time `json:"at"`
	Namespace string    `json:"namespace"`
	Type      string    `json:"type"`
	Reason    string    `json:"reason"`
	Object    string    `json:"object"`
	Message   string    `json:"message"`
	Count     int       `json:"count"`
	Source    string    `json:"source"`
}
