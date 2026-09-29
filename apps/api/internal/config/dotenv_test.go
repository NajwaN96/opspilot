package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEnvFileDoesNotOverrideExplicitValues(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	body := "OPSPILOT_AI_PROVIDER=openai\nOPSPILOT_AI_API_KEY=from-file\nALREADY_SET=from-file\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ALREADY_SET", "from-process")
	t.Setenv("OPSPILOT_AI_PROVIDER", "")
	t.Setenv("OPSPILOT_AI_API_KEY", "")
	os.Unsetenv("OPSPILOT_AI_PROVIDER")
	os.Unsetenv("OPSPILOT_AI_API_KEY")
	loadEnvFile(path)
	if os.Getenv("ALREADY_SET") != "from-process" {
		t.Fatalf("explicit env overridden: %s", os.Getenv("ALREADY_SET"))
	}
	if os.Getenv("OPSPILOT_AI_PROVIDER") != "openai" {
		t.Fatalf("provider %q", os.Getenv("OPSPILOT_AI_PROVIDER"))
	}
	if os.Getenv("OPSPILOT_AI_API_KEY") != "from-file" {
		t.Fatal("missing file value")
	}
}
