package rollout

import "time"

const (
	StatePending   = "PENDING"
	StateRunning   = "RUNNING"
	StateAnalyzing = "ANALYZING"
	StateAwaiting  = "AWAITING_APPROVAL"
	StatePromoting = "PROMOTING"
	StateRolling   = "ROLLING_BACK"
	StateSucceeded = "SUCCEEDED"
	StateFailed    = "FAILED"
	StateAborted   = "ABORTED"
	// StateAttention means the controller could not prove a safe next step.
	// It is not success and it is not an abort.
	StateAttention = "NEEDS_ATTENTION"

	ActionStart   = "START_PAYMENT_API_CANARY"
	ActionAdvance = "ADVANCE_PAYMENT_API_CANARY"
	ActionPromote = "PROMOTE_PAYMENT_API_CANARY"
	ActionAbort   = "ABORT_PAYMENT_API_CANARY"

	ResultPass         = "PASS"
	ResultFail         = "FAIL"
	ResultInsufficient = "INSUFFICIENT_DATA"

	MinRequests = 20
	MaxError    = 0.05
	MaxP95      = 0.300
	MinStage    = 45 * time.Second
	// AnalysisWindow is long enough for 5% of the local ~5 rps generator to reach MinRequests.
	// It is not a caller-supplied range.
	AnalysisWindow = "2m"
)

// Rollout is the persisted canary. Versions are server-owned.
type Rollout struct {
	ID                string    `json:"id"`
	Service           string    `json:"service"`
	Cluster           string    `json:"cluster"`
	Namespace         string    `json:"namespace"`
	StableVersion     string    `json:"stableVersion"`
	CandidateVersion  string    `json:"candidateVersion"`
	CandidateImage    string    `json:"candidateImage"`
	State             string    `json:"state"`
	Weight            int       `json:"weight"`
	StageWeight       int       `json:"stageWeight"`
	LiveWeight        int       `json:"liveWeight"`
	LiveWeightKnown   bool      `json:"liveWeightKnown"`
	CandidateActivity string    `json:"candidateActivity,omitempty"`
	IncidentID        string    `json:"incidentId,omitempty"`
	IncidentStatus    string    `json:"incidentStatus,omitempty"`
	Alerts            []Alert   `json:"alerts,omitempty"`
	Analysis          string    `json:"analysis"`
	ProposalAction    string    `json:"proposalAction"`
	AIAction          string    `json:"aiAction"`
	AISummary         string    `json:"aiSummary"`
	AIStatus          string    `json:"aiStatus"`
	AIMismatch        bool      `json:"aiMismatch"`
	Verification      string    `json:"verification"`
	Requests          float64   `json:"requests"`
	ErrorRate         float64   `json:"errorRate"`
	P95               float64   `json:"p95"`
	StableErrorRate   float64   `json:"stableErrorRate"`
	StableP95         float64   `json:"stableP95"`
	CandidateReady    bool      `json:"candidateReady"`
	StageStarted      time.Time `json:"stageStartedAt"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

type Event struct {
	At     time.Time `json:"at"`
	Title  string    `json:"title"`
	Detail string    `json:"detail"`
	Kind   string    `json:"kind"`
}

type Analysis struct {
	Result   string
	Requests float64
	Errors   float64
	Error    float64
	P95      float64
	Ready    bool
}

// Alert is one allow-listed Alertmanager result plus OpsPilot's reading of it.
// Alertmanager state is never rewritten.
type Alert struct {
	Name           string `json:"name"`
	Alertmanager   string `json:"alertmanager"`
	Interpretation string `json:"interpretation"`
	Note           string `json:"note,omitempty"`
}

// View is the API document. It does not include a kubeconfig or a query.
type View struct {
	Rollout
	Events []Event `json:"events"`
}

// IncidentRef is a detection incident the rollout may be allowed to resolve.
type IncidentRef struct {
	ID        string
	Status    string
	Recovered bool
}
