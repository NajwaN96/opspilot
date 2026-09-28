package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/opspilot/opspilot/apps/api/internal/config"
	"github.com/opspilot/opspilot/apps/api/internal/executor"
	"github.com/opspilot/opspilot/apps/api/internal/httpapi"
	"github.com/opspilot/opspilot/apps/api/internal/kubernetes"
	"github.com/opspilot/opspilot/apps/api/internal/repository"
	"github.com/opspilot/opspilot/apps/api/internal/repository/memory"
	"github.com/opspilot/opspilot/apps/api/internal/repository/postgres"
	"github.com/opspilot/opspilot/apps/api/internal/service"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg := config.Load()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var (
		repo    repository.Catalog
		pg      *postgres.Store
		dbState = "memory"
	)
	if cfg.DatabaseURL != "" {
		store, err := postgres.Open(ctx, postgres.Options{
			DatabaseURL: cfg.DatabaseURL,
			Env:         cfg.Env,
			Step:        cfg.Step,
			Cluster:     cfg.ClusterName,
		})
		if err != nil {
			logger.Error("postgres", "error", err)
			os.Exit(1)
		}
		defer store.Close()
		repo = store
		pg = store
		dbState = "ok"
		logger.Info("postgres ready", "demo_reset", store.DemoResetAllowed())
	} else {
		repo = memory.New(memory.Options{Step: cfg.Step})
		logger.Info("postgres disabled, using in-memory state")
	}

	reader := openKubernetes(cfg, logger)
	svc := service.New(repo, executor.Simulated{Logger: logger})
	svc.SetDemoResetEnabled(cfg.AllowDemoReset && (pg == nil || pg.DemoResetAllowed()))

	if pg != nil {
		go kubernetes.Run(ctx, cfg.SyncInterval, reader, pg, logger)
	}

	api := httpapi.New(svc, logger, cfg.Version)
	api.Kubernetes = reader
	api.Ready = func(r *http.Request) map[string]any {
		database := dbState
		if pg != nil {
			if err := pg.Ping(r.Context()); err != nil {
				database = "down"
			} else {
				database = "ok"
			}
		}
		kubernetesState := "disconnected"
		if status, err := reader.Status(r.Context()); err == nil {
			kubernetesState = status.Connectivity
		}
		return map[string]any{
			"database":         database,
			"kubernetes":       kubernetesState,
			"demoResetEnabled": cfg.AllowDemoReset && (pg == nil || pg.DemoResetAllowed()),
		}
	}
	handler := httpapi.WithCORS(cfg.CORSOrigins, api.Handler())

	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("opspilot api listening", "addr", cfg.Addr, "version", cfg.Version, "env", cfg.Env)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server stopped", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown", "error", err)
		os.Exit(1)
	}
	logger.Info("shutdown complete")
}

func openKubernetes(cfg config.Config, logger *slog.Logger) kubernetes.Reader {
	reader, err := kubernetes.Load(kubernetes.Options{
		Kubeconfig: cfg.Kubeconfig,
		Cluster:    cfg.ClusterName,
		Namespaces: cfg.Namespaces,
	})
	if err != nil {
		logger.Info("kubernetes disconnected", "error", err.Error())
		return kubernetes.Disconnected{
			Cluster:   cfg.ClusterName,
			Namespace: first(cfg.Namespaces),
			Scope:     cfg.Namespaces,
			Message:   "Kubeconfig is not available",
		}
	}
	logger.Info("kubernetes client configured", "cluster", cfg.ClusterName, "namespaces", cfg.Namespaces)
	return reader
}

func first(items []string) string {
	if len(items) == 0 {
		return "demo-shop"
	}
	return items[0]
}
