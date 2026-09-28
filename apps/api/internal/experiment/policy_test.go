package experiment

import "testing"

func TestRealExperimentAllowlist(t *testing.T) {
	policy := Policy{Allow: true}
	if err := policy.Validate(ServiceID, ScenarioPaymentDegraded, 60); err != nil {
		t.Fatal(err)
	}
	if err := policy.Validate("payment-api", ScenarioPaymentDegraded, 60); err == nil {
		t.Fatal("simulated service id must not receive a real experiment")
	}
	if err := policy.Validate("k8s_kube-system_payment-api", ScenarioPaymentDegraded, 60); err == nil {
		t.Fatal("other namespace")
	}
	if err := policy.Validate(ServiceID, "kill-pod", 60); err == nil {
		t.Fatal("scenario")
	}
	if err := policy.Validate(ServiceID, ScenarioPaymentDegraded, 10); err == nil {
		t.Fatal("short duration")
	}
	if err := policy.Validate(ServiceID, ScenarioPaymentDegraded, 300); err == nil {
		t.Fatal("five minutes is not in the UI allowlist")
	}
	if err := (Policy{}).Validate(ServiceID, ScenarioPaymentDegraded, 60); err == nil {
		t.Fatal("production")
	}
}

func TestParametersStayInsideDemoShop(t *testing.T) {
	got := Parameters()
	if got["namespace"] != "demo-shop" || got["cluster"] != "opspilot-dev" || got["service"] != "payment-api" {
		t.Fatal(got)
	}
}
