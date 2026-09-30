package platform

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCatalogLoadsDemoShop(t *testing.T) {
	root, err := FindRoot()
	if err != nil {
		t.Fatal(err)
	}
	catalog, library, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Services) != 5 {
		t.Fatalf("services = %d", len(catalog.Services))
	}
	if _, ok := catalog.Get("payment-api"); !ok {
		t.Fatal("payment-api missing")
	}
	if _, ok := library.Get("bad-release"); !ok {
		t.Fatal("bad-release runbook missing")
	}
	card := Scorecard(catalog)
	if card["source"] != "service-contract" {
		t.Fatalf("source = %v", card["source"])
	}
}

func TestMaterializeRefusesExistingAndWritesPreview(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "platform", "catalog"), 0o755); err != nil {
		t.Fatal(err)
	}
	repo, err := FindRoot()
	if err != nil {
		t.Fatal(err)
	}
	catalog, _, err := Load(repo)
	if err != nil {
		t.Fatal(err)
	}
	if err := copyTree(filepath.Join(repo, "templates"), filepath.Join(root, "templates"), nil); err != nil {
		t.Fatal(err)
	}
	if _, err := Materialize(root, GenerateRequest{Name: "payment-api", Runtime: "go", Owner: "payments", SLO: "99.9", Strategy: "canary"}, catalog); err == nil {
		t.Fatal("expected refusal for an existing service")
	}
	rel, err := Materialize(root, GenerateRequest{Name: "billing-api", Runtime: "go", Owner: "payments", SLO: "99.9", Strategy: "rolling"}, catalog)
	if err != nil {
		t.Fatal(err)
	}
	if rel != "generated/services/billing-api" {
		t.Fatalf("path = %s", rel)
	}
	body, err := os.ReadFile(filepath.Join(root, rel, "deploy", "deployment.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if !strings.Contains(text, "name: billing-api") || strings.Contains(text, "__SERVICE_NAME__") {
		t.Fatalf("substitution failed:\n%s", text)
	}
}
