package httpapi

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/lunararch/oryn/apps/server/internal/api"
	"github.com/lunararch/oryn/apps/server/internal/clients"
)

const (
	heartbeatIntervalSeconds = 30
	sessionTimeoutSeconds    = 120
)

type clientContextKey struct{}

func (o Options) requireClient(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if o.Clients == nil {
			writeError(w, http.StatusServiceUnavailable, "internal_error", "client service is not configured")
			return
		}
		header := r.Header.Get("Authorization")
		token, ok := strings.CutPrefix(header, "Bearer ")
		if !ok || token == "" {
			writeError(w, http.StatusUnauthorized, "unauthorized", "client token is required")
			return
		}
		client, err := o.Clients.Authenticate(r.Context(), token)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized", "client token is invalid or revoked")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), clientContextKey{}, client)))
	})
}

func clientFromContext(ctx context.Context) (clients.Client, bool) {
	client, ok := ctx.Value(clientContextKey{}).(clients.Client)
	return client, ok
}

func (o Options) createPairingCode(w http.ResponseWriter, r *http.Request) {
	if o.Clients == nil {
		writeError(w, http.StatusServiceUnavailable, "internal_error", "client service is not configured")
		return
	}
	code, expires, err := o.Clients.CreatePairingCode(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, api.PairingCodeResponse{
		Code:      code,
		ExpiresAt: formatUnix(expires),
	})
}

func (o Options) registerClient(w http.ResponseWriter, r *http.Request) {
	if o.Clients == nil {
		writeError(w, http.StatusServiceUnavailable, "internal_error", "client service is not configured")
		return
	}
	var request api.RegisterClientRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	client, token, err := o.Clients.Register(r.Context(), request.Code, request.Name, request.Platform)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, api.RegisterClientResponse{
		Client: toDesktopClientDTO(client),
		Token:  token,
	})
}

func (o Options) listClients(w http.ResponseWriter, r *http.Request) {
	if o.Clients == nil {
		writeJSON(w, http.StatusOK, api.ClientsResponse{Items: []api.DesktopClient{}})
		return
	}
	items, err := o.Clients.List(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	out := make([]api.DesktopClient, 0, len(items))
	for _, item := range items {
		out = append(out, toDesktopClientDTO(item))
	}
	writeJSON(w, http.StatusOK, api.ClientsResponse{Items: out})
}

func (o Options) revokeClient(w http.ResponseWriter, r *http.Request) {
	if o.Clients == nil {
		writeError(w, http.StatusServiceUnavailable, "internal_error", "client service is not configured")
		return
	}
	if err := o.Clients.Revoke(r.Context(), chi.URLParam(r, "clientID")); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (o Options) startDesktopSession(w http.ResponseWriter, r *http.Request) {
	client, ok := clientFromContext(r.Context())
	if !ok || o.Sessions == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "client token is required")
		return
	}
	var request api.DesktopSessionRequest
	if err := decodeJSON(w, r, &request); err != nil || request.GameID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "gameId is required")
		return
	}
	session, err := o.Sessions.StartDesktop(r.Context(), client.ID, request.GameID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	dto := toPlaySessionDTO(session)
	dto.HeartbeatIntervalSeconds = heartbeatIntervalSeconds
	dto.TimeoutSeconds = sessionTimeoutSeconds
	writeJSON(w, http.StatusCreated, dto)
}

func (o Options) desktopHeartbeat(w http.ResponseWriter, r *http.Request) {
	client, ok := clientFromContext(r.Context())
	if !ok || o.Sessions == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "client token is required")
		return
	}
	if err := o.Sessions.Heartbeat(r.Context(), client.ID, chi.URLParam(r, "sessionID")); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (o Options) finishDesktopSession(w http.ResponseWriter, r *http.Request) {
	client, ok := clientFromContext(r.Context())
	if !ok || o.Sessions == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "client token is required")
		return
	}
	var request api.FinishSessionRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	session, err := o.Sessions.FinishDesktop(r.Context(), client.ID, chi.URLParam(r, "sessionID"), request.Status, request.ExitCode)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toPlaySessionDTO(session))
}

func toDesktopClientDTO(client clients.Client) api.DesktopClient {
	dto := api.DesktopClient{
		ID:           client.ID,
		Name:         client.Name,
		Platform:     client.Platform,
		RegisteredAt: formatUnix(client.RegisteredAt),
		LastSeenAt:   formatUnix(client.LastSeenAt),
	}
	if client.RevokedAt != nil {
		dto.RevokedAt = formatUnix(*client.RevokedAt)
	}
	return dto
}
