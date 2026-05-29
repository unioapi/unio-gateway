package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/ThankCat/unio-api/internal/bootstrap"
	"github.com/ThankCat/unio-api/internal/platform/config"
	"github.com/ThankCat/unio-api/internal/platform/failure"
	"github.com/ThankCat/unio-api/internal/platform/store"
)

func main() {
	preLogger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		preLogger.Error("load config failed", failure.LogArgs(err)...)
		os.Exit(1)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.Log.Level}))

	startupCtx, startupCancel := context.WithTimeout(context.Background(), cfg.Worker.StartupTimeout)
	defer startupCancel()

	pgPool, err := store.OpenPostgres(startupCtx, cfg.DB)
	if err != nil {
		logger.Error("open postgres failed", failure.LogArgs(err)...)
		os.Exit(1)
	}
	defer pgPool.Close()
	logger.Info("postgres connected")

	app, err := bootstrap.NewWorkerServerApp(startupCtx, bootstrap.WorkerServerAppDeps{
		Logger: logger,
		Config: cfg,
		DB:     pgPool,
	})
	if err != nil {
		logger.Error("worker app failed", failure.LogArgs(err)...)
		os.Exit(1)
	}

	runCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger.Info("worker server starting")
	if err := app.Runner.Run(runCtx); err != nil {
		logger.Error("worker server failed", failure.LogArgs(err)...)
		os.Exit(1)
	}
	logger.Info("worker server stopped")
}
