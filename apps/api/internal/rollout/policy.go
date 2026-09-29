package rollout

import (
	"fmt"

	"github.com/opspilot/opspilot/apps/api/internal/release"
)

// Proposal is the deterministic next action. An AI string cannot change it.
func Proposal(analysis string, weight int, candidate string) (string, bool) {
	switch {
	case analysis == ResultFail:
		return ActionAbort, true
	case analysis == ResultPass && weight == 50 && candidate == release.CandidateGoodVersion:
		return ActionPromote, true
	case analysis == ResultPass && (weight == 5 || weight == 25):
		return ActionAdvance, true
	default:
		return "", false
	}
}

// AllowApprove rejects every transition the SLO gate does not permit.
func AllowApprove(item Rollout, action string) error {
	if item.Namespace != release.Namespace || item.Service != release.Deployment || item.Cluster != release.Cluster {
		return fmt.Errorf("target is not demo-shop/payment-api")
	}
	if item.State != StateAwaiting {
		return fmt.Errorf("rollout is not awaiting approval")
	}
	if action != item.ProposalAction {
		return fmt.Errorf("action does not match the proposal")
	}
	switch action {
	case ActionPromote:
		if item.Analysis != ResultPass || item.Weight != 50 || item.CandidateVersion != release.CandidateGoodVersion {
			return fmt.Errorf("promotion requires a passing 50 percent gate on %s", release.CandidateGoodVersion)
		}
		if !release.Promotable(item.CandidateImage, item.CandidateVersion) {
			return fmt.Errorf("image is not promotable")
		}
	case ActionAbort:
		if item.Analysis != ResultFail {
			return fmt.Errorf("abort requires a failing gate")
		}
	default:
		return fmt.Errorf("action is not executable")
	}
	return nil
}

// AllowStart rejects a second canary and any stable release other than 1.4.2.
func AllowStart(active bool, stableImage, stableVersion, candidateImage, candidateVersion string) error {
	if active {
		return fmt.Errorf("a payment-api rollout is already active")
	}
	if stableImage != release.GoodImage || stableVersion != release.GoodVersion {
		return fmt.Errorf("stable payment-api must be %s", release.GoodVersion)
	}
	if !release.Candidate(candidateImage, candidateVersion) {
		return fmt.Errorf("candidate is not a known payment-api release")
	}
	return nil
}

func nextWeight(weight int) (int, bool) {
	switch weight {
	case 5:
		return 25, true
	case 25:
		return 50, true
	default:
		return 0, false
	}
}

func activeState(state string) bool {
	switch state {
	case StatePending, StateRunning, StateAnalyzing, StateAwaiting, StatePromoting, StateRolling, StateAttention:
		return true
	default:
		return false
	}
}
