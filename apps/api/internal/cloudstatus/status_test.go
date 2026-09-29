package cloudstatus

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestFromProbeMasksAccountAndStaysPlanOnly(t *testing.T) {
	full := "123456789012"
	arn := "arn:aws:iam::" + full + ":user/portfolio"
	status := FromProbe(Probe{Account: full, Region: "us-east-1", Method: "console-login"})
	_ = arn
	if status.AccountState != "connected" {
		t.Fatalf("state %s", status.AccountState)
	}
	if status.AccountMasked != "********9012" {
		t.Fatalf("mask %s", status.AccountMasked)
	}
	if status.Region != "us-east-1" || status.Authentication != "console-login" {
		t.Fatalf("identity display %+v", status)
	}
	if status.CloudDeployment != "PLAN ONLY" || status.EKS != "plan-only" || status.PaidCloudResources != "BLOCKED" {
		t.Fatalf("deployment %+v", status)
	}
	if status.KubernetesRuntime != "LOCAL k3d" || status.IntentionalInfrastructureSpend != "$0" {
		t.Fatalf("runtime %+v", status)
	}
	raw, err := json.Marshal(status)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, secret := range []string{full, arn, "AKIA", "ASIA"} {
		if strings.Contains(text, secret) {
			t.Fatalf("response exposed %s", secret)
		}
	}
}

func TestFromProbeWithoutCredentialsDoesNotPretendToBeConnected(t *testing.T) {
	status := FromProbe(Probe{Err: errors.New("NoCredentials"), Account: "123456789012", Region: "us-east-1"})
	if status.AccountState != "not-connected" || status.AccountMasked != "" || status.Region != "" {
		t.Fatalf("%+v", status)
	}
	if status.CloudDeployment != "PLAN ONLY" || status.EKS != "plan-only" || status.PaidCloudResources != "BLOCKED" {
		t.Fatalf("guard display %+v", status)
	}
}

func TestFromProbeRejectsGovAndPartialIDs(t *testing.T) {
	status := FromProbe(Probe{Account: "1234", Region: "us-gov-west-1", Method: "profile"})
	if status.AccountState != "not-connected" || status.Region != "" {
		t.Fatalf("%+v", status)
	}
	status = FromProbe(Probe{Account: "123456789012", Region: "us-gov-west-1", Method: "profile"})
	if status.AccountState != "connected" || status.Region != "" || status.AccountMasked != "********9012" {
		t.Fatalf("%+v", status)
	}
}
