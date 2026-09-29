package investigate

import "time"

const (
	ActionNone          = "NO_ACTION"
	ActionContinue      = "CONTINUE_INVESTIGATION"
	ActionRollback      = "ROLLBACK_PAYMENT_API"
	ActionStopLab       = "STOP_RELIABILITY_EXPERIMENT"
	StatusNotRequested  = "not_requested"
	StatusQueued        = "queued"
	StatusCollecting    = "collecting_evidence"
	StatusInvestigating = "investigating"
	StatusValidating    = "validating"
	StatusCompleted     = "completed"
	StatusFailed        = "failed"
	StatusInvalid       = "invalid_output"
)

// Item is one bounded piece of evidence. The investigator may cite its ID and nothing else.
type Item struct {
	ID     string    `json:"id"`
	Source string    `json:"source"`
	Kind   string    `json:"kind"`
	At     time.Time `json:"at,omitempty"`
	Title  string    `json:"title"`
	Body   string    `json:"body"`
}

// Snapshot is the only infrastructure view sent to a provider.
type Snapshot struct {
	ID         string    `json:"id"`
	IncidentID string    `json:"incidentId"`
	Version    int       `json:"version"`
	CreatedAt  time.Time `json:"createdAt"`
	Items      []Item    `json:"items"`
	Omitted    []string  `json:"omitted,omitempty"`
	TraceCount int       `json:"traceCount"`
}

type Cause struct {
	Cause                    string   `json:"cause"`
	Confidence               float64  `json:"confidence"`
	SupportingEvidenceIDs    []string `json:"supporting_evidence_ids"`
	ContradictingEvidenceIDs []string `json:"contradicting_evidence_ids"`
}

type Observation struct {
	Statement   string   `json:"statement"`
	EvidenceIDs []string `json:"evidence_ids"`
}

type Action struct {
	ActionType  string   `json:"action_type"`
	Reason      string   `json:"reason"`
	EvidenceIDs []string `json:"evidence_ids"`
}

// Output is the structured investigation. It is not a remediation proposal.
type Output struct {
	Summary            string        `json:"summary"`
	LikelyCauses       []Cause       `json:"likely_causes"`
	Observations       []Observation `json:"observations"`
	RecommendedAction  Action        `json:"recommended_action"`
	MissingInformation []string      `json:"missing_information"`
	RiskNotes          []string      `json:"risk_notes"`
}

type Validation struct {
	Accepted bool     `json:"accepted"`
	Errors   []string `json:"errors,omitempty"`
}

type Usage struct {
	PromptTokens     int `json:"promptTokens"`
	CompletionTokens int `json:"completionTokens"`
}

// View is the API shape. Real is true only for an accepted OpenAI investigation.
type View struct {
	Status             string        `json:"status"`
	Provider           string        `json:"provider"`
	Model              string        `json:"model"`
	Real               bool          `json:"real"`
	SnapshotID         string        `json:"snapshotId,omitempty"`
	SnapshotVersion    int           `json:"snapshotVersion"`
	Summary            string        `json:"summary,omitempty"`
	LikelyCauses       []Cause       `json:"likelyCauses,omitempty"`
	Observations       []Observation `json:"observations,omitempty"`
	RecommendedAction  Action        `json:"recommendedAction"`
	MissingInformation []string      `json:"missingInformation,omitempty"`
	RiskNotes          []string      `json:"riskNotes,omitempty"`
	Validation         Validation    `json:"validation"`
	Evidence           []Item        `json:"evidence,omitempty"`
	Omitted            []string      `json:"omitted,omitempty"`
	Error              string        `json:"error,omitempty"`
	StartedAt          *time.Time    `json:"startedAt,omitempty"`
	CompletedAt        *time.Time    `json:"completedAt,omitempty"`
}

func AllowedAction(action string) bool {
	switch action {
	case ActionNone, ActionContinue, ActionRollback, ActionStopLab:
		return true
	default:
		return false
	}
}
