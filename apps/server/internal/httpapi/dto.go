package httpapi

import (
	"database/sql"
	"time"

	"github.com/lunararch/oryn/apps/server/internal/api"
)

type libraryDTO = api.Library
type librariesResponse = api.LibrariesResponse
type gameDTO = api.Game
type gamesResponse = api.GamesResponse
type updateGameRequest = api.UpdateGameRequest
type statsDTO = api.Stats
type scanStateDTO = api.ScanState
type scanRunDTO = api.ScanRun
type scanRunsResponse = api.ScanRunsResponse
type startScanRequest = api.StartScanRequest
type healthResponse = api.HealthResponse
type versionResponse = api.VersionResponse
type errorResponse = api.ErrorResponse

func formatUnix(value int64) string {
	if value <= 0 {
		return ""
	}
	return time.Unix(value, 0).UTC().Format(time.RFC3339)
}

func formatUnixNull(value sql.NullInt64) string {
	if !value.Valid {
		return ""
	}
	return formatUnix(value.Int64)
}
