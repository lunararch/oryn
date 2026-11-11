package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/lunararch/oryn/apps/server/internal/config"
	"github.com/lunararch/oryn/apps/server/internal/games"
	"github.com/lunararch/oryn/apps/server/internal/httpapi"
	"github.com/lunararch/oryn/apps/server/internal/library"
	"github.com/lunararch/oryn/apps/server/internal/scanner"
	"github.com/lunararch/oryn/apps/server/internal/store"
	"github.com/lunararch/oryn/apps/server/internal/store/db"
)

// App owns the server process lifecycle.
type App struct {
	cfg    config.Config
	logger *slog.Logger
	server *http.Server
	store  *store.Store
	probe  library.Probe
}

// New builds the application, opens the database, and prepares the HTTP handler.
func New(cfg config.Config, logger *slog.Logger, version string) (*App, error) {
	if logger == nil {
		logger = slog.Default()
	}

	s, err := store.Open(cfg.DBPath)
	if err != nil {
		return nil, err
	}

	migrateCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := s.Migrate(migrateCtx); err != nil {
		_ = s.Close()
		return nil, err
	}

	queries := db.New(s.DB)
	now := time.Now
	if err := queries.FailStaleScanRuns(migrateCtx, sql.NullInt64{Int64: now().Unix(), Valid: true}); err != nil {
		_ = s.Close()
		return nil, fmt.Errorf("recover stale scans: %w", err)
	}

	var libraryRecord *db.Library
	if cfg.LibraryRoot != "" {
		record, err := library.Ensure(migrateCtx, queries, cfg.LibraryRoot, "", now)
		if err != nil {
			_ = s.Close()
			return nil, err
		}
		libraryRecord = &record
	}

	probe := library.Probe{
		Root:     cfg.LibraryRoot,
		Sentinel: cfg.LibrarySentinel,
		Timeout:  time.Second,
	}
	gamesService := games.NewService(s, now)
	scanService := scanner.NewService(scanner.ServiceOptions{
		Store:    s,
		MaxDepth: cfg.ScanMaxDepth,
		Sentinel: cfg.LibrarySentinel,
		Now:      now,
		Logger:   logger,
	})
	scanManager := scanner.NewManager(scanService, logger, now)

	handler := httpapi.New(httpapi.Options{
		Version: version,
		Ping:    s.Ping,
		LibraryStatus: func(ctx context.Context) httpapi.LibraryStatus {
			switch probe.Status(ctx) {
			case library.StatusAvailable:
				return httpapi.LibraryAvailable
			case library.StatusUnavailable:
				return httpapi.LibraryUnavailable
			default:
				return httpapi.LibraryNotConfigured
			}
		},
		Library: libraryRecord,
		Games:   gamesService,
		Scans:   scanManager,
		Queries: queries,
	})

	return &App{
		cfg:    cfg,
		logger: logger,
		store:  s,
		probe:  probe,
		server: &http.Server{
			Addr:              cfg.HTTPAddr,
			Handler:           handler,
			ReadHeaderTimeout: 10 * time.Second,
			IdleTimeout:       60 * time.Second,
		},
	}, nil
}

// Handler exposes the HTTP handler for tests and embedded use.
func (a *App) Handler() http.Handler {
	return a.server.Handler
}

// Store exposes the database for integration tests.
func (a *App) Store() *store.Store {
	return a.store
}

// Close releases application resources.
func (a *App) Close() error {
	if a == nil || a.store == nil {
		return nil
	}
	return a.store.Close()
}

// Run serves HTTP until ctx is cancelled, then shuts down gracefully.
func (a *App) Run(ctx context.Context) error {
	listener, err := net.Listen("tcp", a.cfg.HTTPAddr)
	if err != nil {
		return err
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- a.server.Serve(listener)
	}()

	a.logger.Info("http server started", "addr", listener.Addr().String())

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		a.logger.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := a.server.Shutdown(shutdownCtx); err != nil {
			return err
		}
		err := <-errCh
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
