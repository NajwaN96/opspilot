package kubernetes

import "testing"

func TestClassifyKeepsLocalAndCloudApart(t *testing.T) {
	mode, venue, label := Classify("opspilot-dev")
	if mode != "local-kubernetes" || venue != "local" || label != "LOCAL — LIVE" {
		t.Fatalf("local %s %s %s", mode, venue, label)
	}
	mode, venue, label = Classify("opspilot-aws-dev")
	if mode != "eks" || venue != "aws-dev" || label != "AWS — PLAN ONLY" {
		t.Fatalf("aws %s %s %s", mode, venue, label)
	}
	mode, venue, label = Classify("prod-cluster")
	if mode != "unknown" || venue != "unknown" || label != "UNKNOWN" {
		t.Fatalf("unknown %s %s %s", mode, venue, label)
	}
}
