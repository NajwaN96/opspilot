package buildmeta

import "testing"

func TestPublicDropsUnsafeBuildLabels(t *testing.T) {
	previous := []string{Version, Commit, BuiltAt}
	t.Cleanup(func() {
		Version, Commit, BuiltAt = previous[0], previous[1], previous[2]
	})
	Version = "arn:aws:iam::123456789012:role/admin"
	Commit = "not-a-git-sha"
	BuiltAt = "/var/task/secret"

	body := Public("something-else")
	if body["version"] != "dev" || body["gitCommit"] != "unknown" || body["buildTime"] != "unknown" {
		t.Fatalf("unsafe labels leaked: %#v", body)
	}
	if body["environment"] != "local-live" {
		t.Fatalf("environment %#v", body["environment"])
	}
	if _, ok := body["account"]; ok {
		t.Fatal("account must not be returned")
	}
}

func TestPublicKeepsACommitAndTimestamp(t *testing.T) {
	previous := []string{Version, Commit, BuiltAt}
	t.Cleanup(func() {
		Version, Commit, BuiltAt = previous[0], previous[1], previous[2]
	})
	Version = "0.3.0"
	Commit = "dd63e087d68321df4539889b1920949e4903a941"
	BuiltAt = "2026-09-30T08:00:00Z"

	body := Public("local-live")
	if body["gitCommit"] != Commit || body["buildTime"] != BuiltAt || body["version"] != "0.3.0" {
		t.Fatalf("body %#v", body)
	}
}
