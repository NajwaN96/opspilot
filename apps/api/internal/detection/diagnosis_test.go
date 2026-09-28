package detection

import "testing"

func TestDiagnosisPrefersApplicationFaultWhenReplicasAreReady(t *testing.T) {
	got := Diagnose(Facts{Requests: 40, ErrorRate: 0.2, P95: 0.6, Ready: 1, Desired: 1, SlowOrFailed: 2, Version: "1.4.2"})
	if got.LikelyCause != "Application-level degradation in payment-api rather than pod availability failure." {
		t.Fatal(got.LikelyCause)
	}
	if len(got.Supporting) < 2 || len(got.Contradicting) != 1 {
		t.Fatalf("%#v", got)
	}
}

func TestDiagnosisMentionsUnavailableReplicas(t *testing.T) {
	got := Diagnose(Facts{Requests: 40, ErrorRate: 0.2, P95: 0.1, Ready: 0, Desired: 1})
	if got.LikelyCause == "Application-level degradation in payment-api rather than pod availability failure." {
		t.Fatal(got.LikelyCause)
	}
}

func TestSLONeedsVolume(t *testing.T) {
	if ComputeSLO(5, 1).Sufficient {
		t.Fatal("expected insufficient")
	}
	got := ComputeSLO(1000, 10)
	if !got.Sufficient || got.Availability < 0.98 || got.Availability > 0.991 {
		t.Fatalf("%#v", got)
	}
	if got.Window != "15m" || got.Target != 0.999 {
		t.Fatalf("%#v", got)
	}
}
