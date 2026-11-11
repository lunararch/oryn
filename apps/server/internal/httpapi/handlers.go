package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/lunararch/oryn/apps/server/internal/games"
	"github.com/lunararch/oryn/apps/server/internal/scanner"
	"github.com/lunararch/oryn/apps/server/internal/store/db"
)

func (o Options) listGames(w http.ResponseWriter, r *http.Request) {
	if o.Games == nil || o.Library == nil {
		writeError(w, http.StatusServiceUnavailable, "library_not_configured", "library is not configured")
		return
	}
	params, err := parseListParams(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	params.LibraryID = o.Library.ID

	page, err := o.Games.List(r.Context(), params)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	items := make([]gameDTO, 0, len(page.Items))
	for _, game := range page.Items {
		items = append(items, toGameDTO(game))
	}
	writeJSON(w, http.StatusOK, gamesResponse{
		Items:      items,
		Page:       page.Page,
		PageSize:   page.PageSize,
		Total:      page.Total,
		TotalPages: page.TotalPages,
	})
}

func (o Options) getGame(w http.ResponseWriter, r *http.Request) {
	if o.Games == nil {
		writeError(w, http.StatusServiceUnavailable, "library_not_configured", "library is not configured")
		return
	}
	game, err := o.Games.Get(r.Context(), chi.URLParam(r, "gameID"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toGameDTO(game))
}

func (o Options) updateGame(w http.ResponseWriter, r *http.Request) {
	if o.Games == nil {
		writeError(w, http.StatusServiceUnavailable, "library_not_configured", "library is not configured")
		return
	}
	var request updateGameRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	patch := games.MetadataPatch{
		Title:            stringField(request.Title),
		Description:      stringField(request.Description),
		Developer:        stringField(request.Developer),
		Publisher:        stringField(request.Publisher),
		ReleaseDate:      stringField(request.ReleaseDate),
		Rating:           intField(request.Rating),
		Notes:            stringField(request.Notes),
		CompletionStatus: stringField(request.CompletionStatus),
	}
	game, err := o.Games.UpdateMetadata(r.Context(), chi.URLParam(r, "gameID"), patch)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toGameDTO(game))
}

func (o Options) setFavorite(w http.ResponseWriter, r *http.Request) {
	if o.Games == nil {
		writeError(w, http.StatusServiceUnavailable, "library_not_configured", "library is not configured")
		return
	}
	if err := o.Games.SetFavorite(r.Context(), chi.URLParam(r, "gameID"), true); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (o Options) removeFavorite(w http.ResponseWriter, r *http.Request) {
	if o.Games == nil {
		writeError(w, http.StatusServiceUnavailable, "library_not_configured", "library is not configured")
		return
	}
	if err := o.Games.SetFavorite(r.Context(), chi.URLParam(r, "gameID"), false); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (o Options) getStats(w http.ResponseWriter, r *http.Request) {
	if o.Games == nil || o.Library == nil {
		writeError(w, http.StatusServiceUnavailable, "library_not_configured", "library is not configured")
		return
	}
	stats, err := o.Games.GetStats(r.Context(), o.Library.ID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, statsDTO{
		Total:     stats.Total,
		Available: stats.Available,
		Missing:   stats.Missing,
		Favorites: stats.Favorites,
	})
}

func (o Options) listLibraries(w http.ResponseWriter, r *http.Request) {
	items := make([]libraryDTO, 0, 1)
	if o.Library != nil {
		record := *o.Library
		if o.Queries != nil {
			if fresh, err := o.Queries.GetLibrary(r.Context(), record.ID); err == nil {
				record = fresh
			}
		}
		item := libraryDTO{ID: record.ID, Name: record.Name}
		if record.LastScanAt.Valid {
			item.LastScanAt = formatUnix(record.LastScanAt.Int64)
		}
		if o.Games != nil {
			if stats, err := o.Games.GetStats(r.Context(), record.ID); err == nil {
				item.GameCount = stats.Total
			}
		}
		items = append(items, item)
	}
	writeJSON(w, http.StatusOK, librariesResponse{Items: items})
}

func (o Options) startScan(w http.ResponseWriter, r *http.Request) {
	if o.Scans == nil || o.Library == nil {
		writeError(w, http.StatusServiceUnavailable, "library_not_configured", "library is not configured")
		return
	}
	var request startScanRequest
	if err := decodeOptionalJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	id, err := o.Scans.Start(*o.Library, scanner.ScanOptions{AllowMissing: request.AllowMissing})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, scanStateDTO{ID: id, Running: true, Status: scanner.ScanRunning})
}

func (o Options) currentScan(w http.ResponseWriter, r *http.Request) {
	if o.Scans == nil {
		writeJSON(w, http.StatusOK, scanStateDTO{})
		return
	}
	state := o.Scans.Current()
	writeJSON(w, http.StatusOK, toScanStateDTO(state))
}

func (o Options) cancelScan(w http.ResponseWriter, r *http.Request) {
	if o.Scans == nil {
		writeError(w, http.StatusNotFound, "not_found", "scan not found")
		return
	}
	id := chi.URLParam(r, "scanID")
	if !o.Scans.Cancel(id) {
		writeError(w, http.StatusConflict, "scan_not_running", "scan is not running")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (o Options) listScans(w http.ResponseWriter, r *http.Request) {
	if o.Queries == nil {
		writeJSON(w, http.StatusOK, scanRunsResponse{Items: []scanRunDTO{}})
		return
	}
	runs, err := o.Queries.ListScanRuns(r.Context(), 25)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	items := make([]scanRunDTO, 0, len(runs))
	for _, run := range runs {
		items = append(items, toScanRunDTO(run))
	}
	writeJSON(w, http.StatusOK, scanRunsResponse{Items: items})
}

func (o Options) getScan(w http.ResponseWriter, r *http.Request) {
	if o.Queries == nil {
		writeError(w, http.StatusNotFound, "not_found", "scan not found")
		return
	}
	run, err := o.Queries.GetScanRun(r.Context(), chi.URLParam(r, "scanID"))
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "not_found", "scan not found")
		return
	}
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toScanRunDTO(run))
}

func parseListParams(r *http.Request) (games.ListParams, error) {
	query := r.URL.Query()
	page, err := intQuery(query.Get("page"), 1)
	if err != nil {
		return games.ListParams{}, fmt.Errorf("invalid page")
	}
	pageSize, err := intQuery(query.Get("pageSize"), 60)
	if err != nil {
		return games.ListParams{}, fmt.Errorf("invalid pageSize")
	}
	params := games.ListParams{
		Query:        strings.TrimSpace(query.Get("q")),
		Page:         page,
		PageSize:     pageSize,
		Sort:         query.Get("sort"),
		Order:        query.Get("order"),
		Availability: query.Get("availability"),
		Completion:   query.Get("completion"),
	}
	if raw := query.Get("favorite"); raw != "" {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			return games.ListParams{}, fmt.Errorf("invalid favorite filter")
		}
		params.Favorite = &value
	}
	return params, nil
}

func intQuery(raw string, fallback int) (int, error) {
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		return 0, fmt.Errorf("invalid number")
	}
	return value, nil
}

func toGameDTO(game games.Game) gameDTO {
	return gameDTO{
		ID:               game.ID,
		Title:            game.Title,
		Description:      game.Description,
		Developer:        game.Developer,
		Publisher:        game.Publisher,
		ReleaseDate:      game.ReleaseDate,
		Version:          game.Version,
		RelativePath:     game.RelativePath,
		Availability:     game.Availability,
		CompletionStatus: game.CompletionStatus,
		Rating:           game.Rating,
		Notes:            game.Notes,
		Favorite:         game.Favorite,
		AddedAt:          formatUnix(game.AddedAt),
		LastSeenAt:       formatUnix(game.LastSeenAt),
	}
}

func toScanStateDTO(state scanner.ScanState) scanStateDTO {
	dto := scanStateDTO{
		ID:         state.ID,
		Running:    state.Running,
		StartedAt:  formatUnix(state.StartedAt.Unix()),
		Discovered: state.Result.Discovered,
		Added:      state.Result.Added,
		Updated:    state.Result.Updated,
		Unchanged:  state.Result.Unchanged,
		Missing:    state.Result.Missing,
		Errors:     state.Result.Errors,
		Status:     state.Result.Status,
		Summary:    state.Result.Summary,
	}
	if !state.FinishedAt.IsZero() {
		dto.FinishedAt = formatUnix(state.FinishedAt.Unix())
	}
	if state.Running {
		dto.Status = scanner.ScanRunning
	}
	if state.Error != "" {
		dto.Summary = state.Error
	}
	return dto
}

func toScanRunDTO(run db.ScanRun) scanRunDTO {
	return scanRunDTO{
		ID:         run.ID,
		Status:     run.Status,
		StartedAt:  formatUnix(run.StartedAt),
		FinishedAt: formatUnixNull(run.FinishedAt),
		Discovered: int(run.DiscoveredCount),
		Added:      int(run.AddedCount),
		Updated:    int(run.UpdatedCount),
		Unchanged:  int(run.UnchangedCount),
		Missing:    int(run.MissingCount),
		Errors:     int(run.ErrorCount),
		Summary:    run.Summary,
	}
}

func stringField(value *string) games.Field[string] {
	return games.Field[string]{Set: true, Value: value}
}

func intField(value *int) games.Field[int] {
	return games.Field[int]{Set: true, Value: value}
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, games.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "game not found")
	case errors.Is(err, games.ErrValidation):
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
	case errors.Is(err, scanner.ErrScanInProgress):
		writeError(w, http.StatusConflict, "scan_in_progress", "a scan is already running")
	default:
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("invalid JSON body")
	}
	return nil
}

func decodeOptionalJSON(w http.ResponseWriter, r *http.Request, target any) error {
	if r.Body == nil || r.ContentLength == 0 {
		return nil
	}
	return decodeJSON(w, r, target)
}

var _ = context.Background
