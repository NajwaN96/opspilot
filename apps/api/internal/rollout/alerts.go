package rollout

// KnownAlerts are the only payment-api rules OpsPilot interprets.
var KnownAlerts = []string{
	"PaymentApiHighErrorRate",
	"PaymentApiHighLatency",
	"PaymentApiCanaryHighErrorRate",
	"PaymentApiCanaryHighLatency",
}

// InterpretAlert explains a firing rule without changing Alertmanager.
// A firing alert stays firing. Recovering means live canary traffic is already gone.
func InterpretAlert(name, alertmanager string, liveKnown bool, liveWeight, desired int, desiredKnown, terminalSafe bool) (interpretation, note string) {
	switch alertmanager {
	case "inactive":
		return "resolved", ""
	case "firing":
		if terminalSafe && liveKnown && desiredKnown && liveWeight == 0 && desired == 0 {
			return "recovering", "Historical samples are still inside the alert evaluation window; live canary traffic is 0%."
		}
		if !liveKnown || !desiredKnown {
			return "unknown", "Live canary routing is not confirmed, so a firing alert is not described as recovered."
		}
		return "active", ""
	default:
		return "unknown", "Alertmanager state is not available, so this alert is not treated as resolved."
	}
}

func alertViews(firing []string, firingKnown, liveKnown bool, liveWeight, desired int, desiredKnown bool, state string) []Alert {
	firingSet := map[string]bool{}
	for _, name := range firing {
		firingSet[name] = true
	}
	terminalSafe := state == StateAborted || state == StateSucceeded
	out := make([]Alert, 0, len(KnownAlerts))
	for _, name := range KnownAlerts {
		am := "unknown"
		if firingKnown {
			if firingSet[name] {
				am = "firing"
			} else {
				am = "inactive"
			}
		}
		interpretation, note := InterpretAlert(name, am, liveKnown, liveWeight, desired, desiredKnown, terminalSafe)
		out = append(out, Alert{Name: name, Alertmanager: am, Interpretation: interpretation, Note: note})
	}
	return out
}
