package httpapi

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/lunararch/oryn/apps/server/internal/api"
	"github.com/lunararch/oryn/apps/server/internal/sessions"
)

func (o Options) createSession(w http.ResponseWriter, r *http.Request) {
	if o.Sessions == nil {
		writeError(w, http.StatusServiceUnavailable, "library_not_configured", "session service is not configured")
		return
	}
	var request api.SessionRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	started, err := time.Parse(time.RFC3339, request.StartedAt)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "startedAt must be an RFC3339 timestamp")
		return
	}
	ended, err := time.Parse(time.RFC3339, request.EndedAt)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "endedAt must be an RFC3339 timestamp")
		return
	}
	session, err := o.Sessions.CreateManual(r.Context(), chi.URLParam(r, "gameID"), started, ended, request.Notes)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toPlaySessionDTO(session))
}

func (o Options) listSessions(w http.ResponseWriter, r *http.Request) {
	if o.Sessions == nil {
		writeError(w, http.StatusServiceUnavailable, "library_not_configured", "session service is not configured")
		return
	}
	page, err := intQuery(r.URL.Query().Get("page"), 1)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid page")
		return
	}
	pageSize, err := intQuery(r.URL.Query().Get("pageSize"), 20)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid pageSize")
		return
	}
	result, err := o.Sessions.ListForGame(r.Context(), chi.URLParam(r, "gameID"), page, pageSize)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	items := make([]api.PlaySession, 0, len(result.Items))
	for _, session := range result.Items {
		items = append(items, toPlaySessionDTO(session))
	}
	writeJSON(w, http.StatusOK, api.PlaySessionsResponse{
		Items:    items,
		Page:     result.Page,
		PageSize: result.PageSize,
		Total:    result.Total,
	})
}

func (o Options) updateSession(w http.ResponseWriter, r *http.Request) {
	if o.Sessions == nil {
		writeError(w, http.StatusServiceUnavailable, "library_not_configured", "session service is not configured")
		return
	}
	var request api.SessionRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	started, err := time.Parse(time.RFC3339, request.StartedAt)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "startedAt must be an RFC3339 timestamp")
		return
	}
	ended, err := time.Parse(time.RFC3339, request.EndedAt)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "endedAt must be an RFC3339 timestamp")
		return
	}
	session, err := o.Sessions.UpdateManual(r.Context(), chi.URLParam(r, "sessionID"), started, ended, request.Notes)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toPlaySessionDTO(session))
}

func (o Options) deleteSession(w http.ResponseWriter, r *http.Request) {
	if o.Sessions == nil {
		writeError(w, http.StatusServiceUnavailable, "library_not_configured", "session service is not configured")
		return
	}
	if err := o.Sessions.DeleteManual(r.Context(), chi.URLParam(r, "sessionID")); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func toPlaySessionDTO(session sessions.Session) api.PlaySession {
	return api.PlaySession{
		ID:              session.ID,
		GameID:          session.GameID,
		StartedAt:       formatUnix(session.StartedAt),
		EndedAt:         formatUnix(session.EndedAt),
		DurationSeconds: session.DurationSeconds,
		Status:          session.Status,
		Source:          session.Source,
		ExitCode:        session.ExitCode,
		Notes:           session.Notes,
	}
}
