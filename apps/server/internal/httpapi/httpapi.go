package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

// LibraryStatus describes whether the configured game library is currently
// reachable. It is deliberately independent from process health: an
// unavailable SMB mount must not make the API process look unhealthy.
type LibraryStatus string

const (
	LibraryAvailable     LibraryStatus = "available"
	LibraryUnavailable   LibraryStatus = "unavailable"
	LibraryNotConfigured LibraryStatus = "not_configured"
)

// Options contains the dependencies exposed by the HTTP API.
type Options struct {
	Version       string
	Ping          func(context.Context) error
	LibraryStatus func(context.Context) LibraryStatus
}

type healthResponse struct {
	Status   string `json:"status"`
	Version  string `json:"version"`
	Database string `json:"database"`
	Library  string `json:"library"`
}

type versionResponse struct {
	Version string `json:"version"`
}

type errorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// New builds the HTTP API router.
func New(opts Options) http.Handler {
	router := chi.NewRouter()
	router.Use(requestID)
	router.Use(middleware.Recoverer)
	router.Use(securityHeaders)
	router.Use(middleware.Timeout(30 * time.Second))

	router.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", healthHandler(opts))
		r.Get("/version", versionHandler(opts))
	})

	router.NotFound(func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, "not_found", "not found")
	})
	router.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	})

	return router
}

func healthHandler(opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response := healthResponse{
			Status:   "ok",
			Version:  opts.Version,
			Database: "not_configured",
			Library:  string(LibraryNotConfigured),
		}

		statusCode := http.StatusOK
		if opts.Ping != nil {
			if err := opts.Ping(r.Context()); err != nil {
				response.Status = "degraded"
				response.Database = "unavailable"
				statusCode = http.StatusServiceUnavailable
			} else {
				response.Database = "ok"
			}
		}

		if opts.LibraryStatus != nil {
			response.Library = string(opts.LibraryStatus(r.Context()))
		}

		writeJSON(w, statusCode, response)
	}
}

func versionHandler(opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, versionResponse{Version: opts.Version})
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	response := errorResponse{}
	response.Error.Code = code
	response.Error.Message = message
	writeJSON(w, status, response)
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}

func requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := uuid.NewString()
		w.Header().Set("X-Request-Id", id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey{}, id)))
	})
}

type requestIDKey struct{}
