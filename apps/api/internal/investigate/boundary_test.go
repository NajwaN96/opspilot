package investigate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInvestigatorPackageHasNoExecutionImports(t *testing.T) {
	dir := "."
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	banned := []string{"internal/executor", "internal/kubernetes", "os/exec", "database/sql", "docker"}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		body, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		for _, word := range banned {
			if strings.Contains(string(body), `"`+word) || strings.Contains(string(body), word+"/") {
				t.Fatalf("%s imports %s", entry.Name(), word)
			}
		}
	}
}
