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
	"github.com/opspilot/opspilot/apps/api/internal/repository/memory"
	"github.com/opspilot/opspilot/apps/api/internal/service"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg := config.Load()
	store := memory.New(memory.Options{Step: cfg.Step})
	svc := service.New(store, executor.Simulated{Logger: logger})
	api := httpapi.New(svc, logger, cfg.Version)
	handler := httpapi.WithCORS(cfg.CORSOrigins, api.Handler())

	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("opspilot api listening", "addr", cfg.Addr, "version", cfg.Version)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server stopped", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("shutdown", "error", err)
		os.Exit(1)
	}
}
