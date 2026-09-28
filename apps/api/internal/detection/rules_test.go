package detection

import "testing"

func TestEvaluateRequiresVolume(t *testing.T) {
	got := Evaluate(Sample{OK: true, Requests: 3, Errors: 3, P95: 2})
	if got.Breach || got.Healthy {
		t.Fatalf("%#v", got)
	}
}

func TestEvaluateFiresOnErrorOrLatency(t *testing.T) {
	errors := Evaluate(Sample{OK: true, Requests: 40, Errors: 3, P95: 0.1})
	if !errors.Breach {
		t.Fatalf("error rate %#v", errors)
	}
	latency := Evaluate(Sample{OK: true, Requests: 40, Errors: 0, P95: 0.5})
	if !latency.Breach {
		t.Fatalf("latency %#v", latency)
	}
	ok := Evaluate(Sample{OK: true, Requests: 40, Errors: 1, P95: 0.2})
	if !ok.Healthy {
		t.Fatalf("healthy %#v", ok)
	}
}

func TestMissingTelemetryDoesNotFire(t *testing.T) {
	got := Evaluate(Sample{})
	if got.Breach || got.Healthy {
		t.Fatalf("%#v", got)
	}
	streak, action := (Streak{}).Next(got)
	if action != "hold" || streak.Breach != 0 {
		t.Fatalf("%s %#v", action, streak)
	}
}

func TestSustainAndRecover(t *testing.T) {
	breach := Decision{Breach: true}
	streak, action := (Streak{}).Next(breach)
	if action != "pending" {
		t.Fatal(action)
	}
	streak, action = streak.Next(breach)
	if action != "open" || streak.Breach != 2 {
		t.Fatalf("%s %#v", action, streak)
	}
	healthy := Decision{Healthy: true}
	streak, action = streak.Next(healthy)
	if action != "pending" || streak.Breach != 0 {
		t.Fatalf("first healthy %s %#v", action, streak)
	}
	_, action = streak.Next(healthy)
	if action != "recovered" {
		t.Fatal(action)
	}
}

func TestFingerprint(t *testing.T) {
	if Fingerprint("opspilot-dev") != "PAYMENT_API_RELIABILITY_DEGRADATION|payment-api|demo-shop|opspilot-dev" {
		t.Fatal(Fingerprint("opspilot-dev"))
	}
}
