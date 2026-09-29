package rollout

import (
	"strings"
	"testing"
)

func TestFiringCanaryAlertIsRecoveringWhenTrafficIsGone(t *testing.T) {
	kind, note := InterpretAlert("PaymentApiCanaryHighErrorRate", "firing", true, 0, 0, true, true)
	if kind != "recovering" || note == "" {
		t.Fatalf("%s %q", kind, note)
	}
	if !strings.Contains(note, "live canary traffic is 0%") {
		t.Fatal(note)
	}
}

func TestFiringAlertStaysActiveWhileCanaryServes(t *testing.T) {
	kind, note := InterpretAlert("PaymentApiCanaryHighLatency", "firing", true, 5, 1, true, false)
	if kind != "active" || note != "" {
		t.Fatalf("%s %q", kind, note)
	}
}

func TestInactiveAlertIsResolved(t *testing.T) {
	kind, note := InterpretAlert("PaymentApiHighErrorRate", "inactive", true, 0, 0, true, true)
	if kind != "resolved" || note != "" {
		t.Fatalf("%s %q", kind, note)
	}
}

func TestUnknownAlertmanagerIsNotResolved(t *testing.T) {
	kind, note := InterpretAlert("PaymentApiCanaryHighErrorRate", "unknown", true, 0, 0, true, true)
	if kind != "unknown" || note == "" {
		t.Fatalf("%s %q", kind, note)
	}
}

func TestFiringWithoutLiveWeightIsNotRecovering(t *testing.T) {
	kind, _ := InterpretAlert("PaymentApiCanaryHighErrorRate", "firing", false, 0, 0, true, true)
	if kind != "unknown" {
		t.Fatal(kind)
	}
}
