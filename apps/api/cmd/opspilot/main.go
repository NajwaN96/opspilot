package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/opspilot/opspilot/apps/api/internal/config"
	"github.com/opspilot/opspilot/apps/api/internal/detection"
	"github.com/opspilot/opspilot/apps/api/internal/executor"
	"github.com/opspilot/opspilot/apps/api/internal/experiment"
	"github.com/opspilot/opspilot/apps/api/internal/httpapi"
	"github.com/opspilot/opspilot/apps/api/internal/investigate"
	"github.com/opspilot/opspilot/apps/api/internal/kubernetes"
	"github.com/opspilot/opspilot/apps/api/internal/repository"
	"github.com/opspilot/opspilot/apps/api/internal/repository/memory"
	"github.com/opspilot/opspilot/apps/api/internal/repository/postgres"
	"github.com/opspilot/opspilot/apps/api/internal/service"
	"github.com/opspilot/opspilot/apps/api/internal/telemetry"
	"github.com/opspilot/opspilot/apps/api/internal/verify"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	config.LoadLocalEnv()
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

	reader, kubeClient := openKubernetes(cfg, logger)
	sources := telemetrySources(kubeClient)
	svc := service.New(repo, executor.Simulated{Logger: logger})
	svc.SetDemoResetEnabled(cfg.AllowDemoReset && (pg == nil || pg.DemoResetAllowed()))
	svc.SetRolloutsEnabled(cfg.AllowExperiments && kubeClient != nil)
	if kubeClient != nil && pg != nil {
		watcher := &verify.Watcher{
			Metrics: sources.Prometheus,
			Image: func(ctx context.Context) (string, error) {
				image, _, err := kubeClient.CurrentPayment(ctx)
				return image, err
			},
			Store:  pg,
			Logger: logger,
		}
		svc.SetPaymentRollback(kubeClient, watcher)
		watcher.Resume(ctx)
	}
	if pg != nil && kubeClient != nil && cfg.AllowExperiments {
		lab := &experiment.Controller{
			Policy: experiment.Policy{Allow: true},
			Token:  cfg.FaultToken,
			Fault:  kubeClient,
			Store:  pg,
		}
		svc.SetLab(lab)
		go runEvery(ctx, 5*time.Second, func() {
			if err := lab.Expire(ctx); err != nil {
				logger.Error("experiment expire", "error", err)
			}
		})
	}
	if pg != nil {
		go kubernetes.Run(ctx, cfg.SyncInterval, reader, pg, logger)
		engine := &detection.Engine{
			Metrics:     sources.Prometheus,
			Traces:      sources.Jaeger,
			Cluster:     reader,
			Sink:        pg,
			ClusterName: cfg.ClusterName,
		}
		go runEvery(ctx, detection.EvalInterval, func() {
			if err := engine.Tick(ctx); err != nil {
				logger.Error("detection", "error", err)
			}
		})
	}

	api := httpapi.New(svc, logger, cfg.Version)
	api.Kubernetes = reader
	api.Sources = sources
	var investigationStore investigate.Store
	if pg != nil {
		investigationStore = pg
	}
	investigator := &investigate.Runner{
		Provider: aiProvider(cfg, logger),
		Reader: investigate.Reader{
			Metrics: sources.Prometheus,
			Traces:  sources.Jaeger,
			Cluster: reader,
		},
		Store:  investigationStore,
		Logger: logger,
	}
	api.Investigator = investigator
	if pg != nil {
		go runEvery(ctx, 15*time.Second, func() {
			if err := investigator.Tick(ctx); err != nil {
				logger.Error("investigation tick", "error", err)
			}
		})
	}
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
		telemetryState := sources.ComponentStatus(r.Context())
		return map[string]any{
			"database":         database,
			"kubernetes":       kubernetesState,
			"demoResetEnabled": cfg.AllowDemoReset && (pg == nil || pg.DemoResetAllowed()),
			"prometheus":       telemetryState["prometheus"],
			"opentelemetry":    telemetryState["opentelemetry"],
			"traces":           telemetryState["traces"],
			"ai":               investigator.Status(r.Context()),
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

func aiProvider(cfg config.Config, logger *slog.Logger) investigate.Provider {
	switch cfg.AIProvider {
	case "openai":
		if cfg.AIAPIKey == "" {
			logger.Info("ai investigator unconfigured")
			return investigate.Disabled{}
		}
		logger.Info("ai investigator configured", "provider", "openai", "model", cfg.AIModel, "key", "present")
		return investigate.OpenAI{ModelName: cfg.AIModel, APIKey: cfg.AIAPIKey}
	case "fixture":
		logger.Info("ai investigator fixture")
		return investigate.Fixture{}
	default:
		logger.Info("ai investigator disabled")
		return investigate.Disabled{}
	}
}

func openKubernetes(cfg config.Config, logger *slog.Logger) (kubernetes.Reader, *kubernetes.Client) {
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
		}, nil
	}
	logger.Info("kubernetes client configured", "cluster", cfg.ClusterName, "namespaces", cfg.Namespaces)
	client, _ := reader.(*kubernetes.Client)
	return reader, client
}

func telemetrySources(client *kubernetes.Client) *telemetry.Sources {
	if client == nil {
		return &telemetry.Sources{}
	}
	return &telemetry.Sources{
		Prometheus: telemetry.Prometheus{Get: proxyGetter{client: client, namespace: "opspilot-system", service: "prometheus", port: 9090}},
		Jaeger:     telemetry.Jaeger{Get: proxyGetter{client: client, namespace: "opspilot-system", service: "jaeger", port: 16686}},
		CollectorReady: func(ctx context.Context) error {
			code, _, err := client.ReadProxy(ctx, "opspilot-system", "otel-collector", 13133, "/", nil)
			if err != nil {
				return err
			}
			if code != http.StatusOK {
				return errStatus(code)
			}
			return nil
		},
	}
}

type proxyGetter struct {
	client    *kubernetes.Client
	namespace string
	service   string
	port      int
}

func (p proxyGetter) Get(ctx context.Context, path string, query url.Values) (int, []byte, error) {
	return p.client.ReadProxy(ctx, p.namespace, p.service, p.port, path, query)
}

type errStatus int

func (e errStatus) Error() string { return "unexpected status" }

func runEvery(ctx context.Context, every time.Duration, fn func()) {
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			fn()
		}
	}
}

func first(items []string) string {
	if len(items) == 0 {
		return "demo-shop"
	}
	return items[0]
}
