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
	ID               string    `json:"id"`
	Service          string    `json:"service"`
	Cluster          string    `json:"cluster"`
	Namespace        string    `json:"namespace"`
	StableVersion    string    `json:"stableVersion"`
	CandidateVersion string    `json:"candidateVersion"`
	CandidateImage   string    `json:"candidateImage"`
	State            string    `json:"state"`
	Weight           int       `json:"weight"`
	Analysis         string    `json:"analysis"`
	ProposalAction   string    `json:"proposalAction"`
	AIAction         string    `json:"aiAction"`
	AISummary        string    `json:"aiSummary"`
	AIStatus         string    `json:"aiStatus"`
	AIMismatch       bool      `json:"aiMismatch"`
	Verification     string    `json:"verification"`
	Requests         float64   `json:"requests"`
	ErrorRate        float64   `json:"errorRate"`
	P95              float64   `json:"p95"`
	StableErrorRate  float64   `json:"stableErrorRate"`
	StableP95        float64   `json:"stableP95"`
	CandidateReady   bool      `json:"candidateReady"`
	StageStarted     time.Time `json:"stageStartedAt"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
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

// View is the API document. It does not include a kubeconfig or a query.
type View struct {
	Rollout
	Events []Event `json:"events"`
}
