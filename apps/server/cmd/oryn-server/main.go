package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/lunararch/oryn/apps/server/internal/app"
	"github.com/lunararch/oryn/apps/server/internal/config"
	"github.com/lunararch/oryn/apps/server/internal/store"
)

var version = "dev"

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "backup":
			if err := runBackup(os.Args[2:]); err != nil {
				fmt.Fprintln(os.Stderr, "backup failed:", err)
				os.Exit(1)
			}
			return
		case "version":
			fmt.Println(version)
			return
		default:
			fmt.Fprintf(os.Stderr, "unknown command %q\nusage: oryn-server [backup [path]|version]\n", os.Args[1])
			os.Exit(2)
		}
	}
	if err := runServer(); err != nil {
		fmt.Fprintln(os.Stderr, "server failed:", err)
		os.Exit(1)
	}
}

func runServer() error {
	cfg, err := config.LoadFromEnv()
	if err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}
	logger := newLogger(cfg)
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	application, err := app.New(cfg, logger, version)
	if err != nil {
		return err
	}
	defer func() { _ = application.Close() }()
	return application.Run(ctx)
}

func runBackup(args []string) error {
	cfg, err := config.LoadFromEnv()
	if err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}
	if info, err := os.Stat(cfg.DBPath); err != nil || info.IsDir() {
		return fmt.Errorf("database does not exist: %s", cfg.DBPath)
	}
	destination := ""
	if len(args) > 0 {
		destination = args[0]
	} else {
		destination = filepath.Join(cfg.DataDir, "backups", "oryn-"+time.Now().UTC().Format("20060102-150405")+".db")
	}
	s, err := store.Open(cfg.DBPath)
	if err != nil {
		return err
	}
	defer s.Close()
	if err := s.Backup(context.Background(), destination); err != nil {
		return err
	}
	fmt.Println(destination)
	return nil
}

func newLogger(cfg config.Config) *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
}
