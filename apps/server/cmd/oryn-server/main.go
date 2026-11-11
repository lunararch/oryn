package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/lunararch/oryn/apps/server/internal/app"
	"github.com/lunararch/oryn/apps/server/internal/config"
)

var version = "dev"

func main() {
	cfg, err := config.LoadFromEnv()
	if err != nil {
		slog.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.LogLevel,
	}))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	application := app.New(cfg, logger, version)
	if err := application.Run(ctx); err != nil {
		logger.Error("server stopped with error", "error", err)
		os.Exit(1)
	}
}
