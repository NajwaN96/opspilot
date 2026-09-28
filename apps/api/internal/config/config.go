package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Addr             string
	CORSOrigins      []string
	Step             time.Duration
	Version          string
	DatabaseURL      string
	Env              string
	Kubeconfig       string
	ClusterName      string
	Namespaces       []string
	SyncInterval     time.Duration
	AllowDemoReset   bool
	AllowExperiments bool
	FaultToken       string
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
	env := os.Getenv("OPSPILOT_ENV")
	if env == "" {
		env = "development"
	}
	cluster := os.Getenv("OPSPILOT_K8S_CLUSTER")
	if cluster == "" {
		cluster = "opspilot-dev"
	}
	namespaces := []string{"demo-shop"}
	if raw := os.Getenv("OPSPILOT_K8S_NAMESPACES"); raw != "" {
		namespaces = split(raw)
	}
	syncEvery := 15 * time.Second
	if raw := os.Getenv("OPSPILOT_SYNC_INTERVAL"); raw != "" {
		if parsed, err := time.ParseDuration(raw); err == nil && parsed > 0 {
			syncEvery = parsed
		}
	}
	token := os.Getenv("OPSPILOT_FAULT_TOKEN")
	if token == "" && env != "production" {
		token = "opspilot-local-fault"
	}
	return Config{
		Addr:             addr,
		CORSOrigins:      origins,
		Step:             time.Duration(stepMS) * time.Millisecond,
		Version:          "0.3.0",
		DatabaseURL:      os.Getenv("OPSPILOT_DATABASE_URL"),
		Env:              env,
		Kubeconfig:       os.Getenv("OPSPILOT_KUBECONFIG"),
		ClusterName:      cluster,
		Namespaces:       namespaces,
		SyncInterval:     syncEvery,
		AllowDemoReset:   env != "production",
		AllowExperiments: env != "production",
		FaultToken:       token,
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
