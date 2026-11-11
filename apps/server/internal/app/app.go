package app

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/lunararch/oryn/apps/server/internal/config"
	"github.com/lunararch/oryn/apps/server/internal/httpapi"
	"github.com/lunararch/oryn/apps/server/internal/library"
)

// App owns the server process lifecycle.
type App struct {
	cfg    config.Config
	logger *slog.Logger
	server *http.Server
	probe  library.Probe
}

// New builds the application and its HTTP handler.
func New(cfg config.Config, logger *slog.Logger, version string) *App {
	if logger == nil {
		logger = slog.Default()
	}

	probe := library.Probe{
		Root:     cfg.LibraryRoot,
		Sentinel: cfg.LibrarySentinel,
		Timeout:  time.Second,
	}

	handler := httpapi.New(httpapi.Options{
		Version: version,
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
	})

	return &App{
		cfg:    cfg,
		logger: logger,
		probe:  probe,
		server: &http.Server{
			Addr:              cfg.HTTPAddr,
			Handler:           handler,
			ReadHeaderTimeout: 10 * time.Second,
			IdleTimeout:       60 * time.Second,
		},
	}
}

// Handler exposes the HTTP handler for tests and embedded use.
func (a *App) Handler() http.Handler {
	return a.server.Handler
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
