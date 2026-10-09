package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/mmmIlia/badgoodtune/internal/app"
	"github.com/mmmIlia/badgoodtune/internal/config"
	"github.com/mmmIlia/badgoodtune/internal/logging"
)

func main() {
	if err := run(); err != nil {
		logging.New(os.Stdout, "info").Error("server failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func run() error {
	settings, err := config.Load()
	if err != nil {
		return err
	}
	logger := logging.New(os.Stdout, settings.Logging.Level)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	application, err := app.New(ctx, settings.Database, settings.Auth, settings.HTTP, logger)
	if err != nil {
		return err
	}
	return application.Run(ctx)
}
