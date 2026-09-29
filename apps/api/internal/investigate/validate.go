package investigate

import (
	"encoding/json"
	"regexp"
	"strings"
)

var (
	versionPattern = regexp.MustCompile(`\b\d+\.\d+\.\d+(?:-bad)?\b`)
	tracePattern   = regexp.MustCompile(`\b[0-9a-f]{16,32}\b`)
	deployPattern  = regexp.MustCompile(`\b[a-z0-9]+(?:-[a-z0-9]+)*-api\b`)
)

// Validate checks model output against the snapshot. It does not call a provider or a cluster.
func Validate(snapshot Snapshot, raw []byte) (Output, Validation) {
	var out Output
	if err := json.Unmarshal(raw, &out); err != nil {
		return Output{}, Validation{Accepted: false, Errors: []string{"malformed json"}}
	}
	var errors []string
	if strings.TrimSpace(out.Summary) == "" {
		errors = append(errors, "summary is empty")
	}
	if !AllowedAction(out.RecommendedAction.ActionType) {
		errors = append(errors, "unsupported action")
	}
	known := map[string]Item{}
	var corpus strings.Builder
	for _, item := range snapshot.Items {
		known[item.ID] = item
		corpus.WriteString(item.Body)
		corpus.WriteString(" ")
		corpus.WriteString(item.Title)
		corpus.WriteString(" ")
	}
	checkIDs := func(ids []string, label string) {
		if len(ids) == 0 && label == "cause" {
			errors = append(errors, "likely cause has no supporting evidence")
		}
		for _, id := range ids {
			if _, ok := known[id]; !ok {
				errors = append(errors, "unknown evidence id "+id)
			}
		}
	}
	for _, cause := range out.LikelyCauses {
		if cause.Confidence < 0 || cause.Confidence > 1 {
			errors = append(errors, "confidence is outside 0 to 1")
		}
		checkIDs(cause.SupportingEvidenceIDs, "cause")
		checkIDs(cause.ContradictingEvidenceIDs, "contradicting")
	}
	for _, observation := range out.Observations {
		checkIDs(observation.EvidenceIDs, "observation")
	}
	checkIDs(out.RecommendedAction.EvidenceIDs, "action")
	text := out.Summary + " " + out.RecommendedAction.Reason
	for _, cause := range out.LikelyCauses {
		text += " " + cause.Cause
	}
	for _, version := range versionPattern.FindAllString(text, -1) {
		if !strings.Contains(corpus.String(), version) {
			errors = append(errors, "unknown version "+version)
		}
	}
	for _, id := range tracePattern.FindAllString(text, -1) {
		if !strings.Contains(corpus.String(), id) {
			errors = append(errors, "unknown trace id "+id)
		}
	}
	if strings.Contains(strings.ToLower(text), "kube-system") {
		errors = append(errors, "unknown namespace")
	}
	for _, name := range deployPattern.FindAllString(text, -1) {
		if !strings.Contains(corpus.String(), name) {
			errors = append(errors, "unknown deployment "+name)
		}
	}
	out.Summary = clip(out.Summary, 2000)
	return out, Validation{Accepted: len(errors) == 0, Errors: errors}
}
