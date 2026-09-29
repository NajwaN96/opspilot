package investigate

import (
	"os"
	"path/filepath"
	"strings"
)

func RunbookExcerpt(rule, service string) string {
	if service != "payment-api" && rule != "PAYMENT_API_RELIABILITY_DEGRADATION" {
		return "No runbook is indexed for this service."
	}
	var parts []string
	for _, name := range []string{"payment-api-reliability-degradation.md", "payment-api-rollback.md"} {
		if text := readRunbook(name); text != "" {
			parts = append(parts, text)
		}
	}
	if len(parts) == 0 {
		parts = append(parts, "Reliability degradation on payment-api is an application signal when replicas stay ready. The lab fault is in-process and expires. The bad release is 1.5.0-bad. The only approved rollback target is 1.4.2 on demo-shop/payment-api. Stopping an experiment does not change a Deployment.")
	}
	return clip(strings.Join(parts, "\n"), 1500)
}

func readRunbook(name string) string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for i := 0; i < 6; i++ {
		path := filepath.Join(dir, "runbooks", name)
		body, err := os.ReadFile(path)
		if err == nil {
			return string(body)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
	return ""
}
