package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Addr        string
	CORSOrigins []string
	Step        time.Duration
	Version     string
}

func Load() Config {
	stepMS := 1400
	if raw := os.Getenv("OPSPILOT_REMEDIATION_STEP_MS"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed >= 0 {
			stepMS = parsed
		}
	}
	origins := []string{"http://127.0.0.1:3461", "http://localhost:3461"}
	if raw := os.Getenv("OPSPILOT_CORS_ORIGINS"); raw != "" {
		origins = split(raw)
	}
	addr := os.Getenv("OPSPILOT_ADDR")
	if addr == "" {
		addr = ":8094"
	}
	return Config{
		Addr:        addr,
		CORSOrigins: origins,
		Step:        time.Duration(stepMS) * time.Millisecond,
		Version:     "0.1.0",
	}
}

func split(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
