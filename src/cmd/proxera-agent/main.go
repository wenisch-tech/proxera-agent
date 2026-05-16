package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/wenisch-tech/proxera-agent/internal/config"
	"github.com/wenisch-tech/proxera-agent/internal/logging"
	"github.com/wenisch-tech/proxera-agent/internal/tunnel"
	"github.com/wenisch-tech/proxera-agent/internal/version"
)

func main() {
	cfg, err := config.Load(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "configuration error: %v\n", err)
		os.Exit(2)
	}

	logger := logging.New(cfg.LogLevel)
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	logger.Info("starting proxera client",
		slog.String("version", version.Version),
		slog.String("commit", version.Commit),
		slog.String("buildDate", version.BuildDate),
		slog.String("serverURL", cfg.ServerURL),
	)

	client := tunnel.NewClient(cfg, logger)
	if err := client.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("client stopped with error", slog.Any("error", err))
		os.Exit(1)
	}

	logger.Info("proxera client stopped")
}
