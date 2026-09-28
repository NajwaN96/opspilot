package fault

import (
	"testing"
	"time"
)

func TestFaultExpires(t *testing.T) {
	var state State
	now := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	state.Enable(now.Add(time.Minute), 40, 500*time.Millisecond)
	if !state.Active(now.Add(30 * time.Second)) {
		t.Fatal("expected active")
	}
	if state.Active(now.Add(time.Minute)) {
		t.Fatal("expected expired at the deadline")
	}
	state.Disable()
	if state.Active(now) {
		t.Fatal("expected disabled")
	}
}

func TestFaultClamps(t *testing.T) {
	var state State
	state.Enable(time.Now().Add(time.Second), 140, -1)
	if state.ErrorPercent != 100 || state.Latency != 0 {
		t.Fatalf("%d %s", state.ErrorPercent, state.Latency)
	}
}
