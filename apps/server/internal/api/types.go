// Package api contains the HTTP contract shared with the web and desktop clients.
package api

type Library struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	GameCount  int    `json:"gameCount"`
	LastScanAt string `json:"lastScanAt,omitempty"`
}

type LibrariesResponse struct {
	Items []Library `json:"items"`
}

type Game struct {
	ID               string `json:"id"`
	Title            string `json:"title"`
	Description      string `json:"description"`
	Developer        string `json:"developer"`
	Publisher        string `json:"publisher"`
	ReleaseDate      string `json:"releaseDate"`
	Version          string `json:"version"`
	RelativePath     string `json:"relativePath"`
	Availability     string `json:"availability"`
	CompletionStatus string `json:"completionStatus"`
	Rating           *int   `json:"rating,omitempty"`
	Notes            string `json:"notes"`
	Favorite         bool   `json:"favorite"`
	CoverArtworkID   string `json:"coverArtworkId,omitempty"`
	BannerArtworkID  string `json:"bannerArtworkId,omitempty"`
	AddedAt          string `json:"addedAt"`
	LastSeenAt       string `json:"lastSeenAt"`
	LastPlayedAt     string `json:"lastPlayedAt,omitempty"`
	PlaytimeSeconds  int    `json:"playtimeSeconds"`
}

type GamesResponse struct {
	Items      []Game `json:"items"`
	Page       int    `json:"page"`
	PageSize   int    `json:"pageSize"`
	Total      int    `json:"total"`
	TotalPages int    `json:"totalPages"`
}

type UpdateGameRequest struct {
	Title            *string `json:"title"`
	Description      *string `json:"description"`
	Developer        *string `json:"developer"`
	Publisher        *string `json:"publisher"`
	ReleaseDate      *string `json:"releaseDate"`
	Rating           *int    `json:"rating"`
	Notes            *string `json:"notes"`
	CompletionStatus *string `json:"completionStatus"`
}

type Stats struct {
	Total           int `json:"total"`
	Available       int `json:"available"`
	Missing         int `json:"missing"`
	Favorites       int `json:"favorites"`
	PlayedGames     int `json:"playedGames"`
	PlaytimeSeconds int `json:"playtimeSeconds"`
}

type PlaySession struct {
	ID                       string `json:"id"`
	GameID                   string `json:"gameId"`
	ClientID                 string `json:"clientId,omitempty"`
	StartedAt                string `json:"startedAt"`
	EndedAt                  string `json:"endedAt,omitempty"`
	DurationSeconds          int    `json:"durationSeconds"`
	Status                   string `json:"status"`
	Source                   string `json:"source"`
	ExitCode                 *int   `json:"exitCode,omitempty"`
	Notes                    string `json:"notes"`
	LastHeartbeatAt          string `json:"lastHeartbeatAt,omitempty"`
	HeartbeatIntervalSeconds int    `json:"heartbeatIntervalSeconds,omitempty"`
	TimeoutSeconds           int    `json:"timeoutSeconds,omitempty"`
}

type PlaySessionsResponse struct {
	Items    []PlaySession `json:"items"`
	Page     int           `json:"page"`
	PageSize int           `json:"pageSize"`
	Total    int           `json:"total"`
}

type SessionRequest struct {
	StartedAt string `json:"startedAt"`
	EndedAt   string `json:"endedAt"`
	Notes     string `json:"notes"`
}

type Collection struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	GameCount   int    `json:"gameCount"`
}

type CollectionsResponse struct {
	Items []Collection `json:"items"`
}

type CollectionMembership struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	GameCount   int    `json:"gameCount"`
	Contains    bool   `json:"contains"`
}

type CollectionMembershipsResponse struct {
	Items []CollectionMembership `json:"items"`
}

type CollectionRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type DesktopClient struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Platform     string `json:"platform"`
	RegisteredAt string `json:"registeredAt"`
	LastSeenAt   string `json:"lastSeenAt"`
	RevokedAt    string `json:"revokedAt,omitempty"`
}

type ClientsResponse struct {
	Items []DesktopClient `json:"items"`
}

type PairingCodeResponse struct {
	Code      string `json:"code"`
	ExpiresAt string `json:"expiresAt"`
}

type RegisterClientRequest struct {
	Code     string `json:"code"`
	Name     string `json:"name"`
	Platform string `json:"platform"`
}

type RegisterClientResponse struct {
	Client DesktopClient `json:"client"`
	Token  string        `json:"token"`
}

type DesktopSessionRequest struct {
	GameID string `json:"gameId"`
}

type FinishSessionRequest struct {
	Status   string `json:"status"`
	ExitCode *int   `json:"exitCode"`
}

type ScanState struct {
	ID         string `json:"id,omitempty"`
	Running    bool   `json:"running"`
	StartedAt  string `json:"startedAt,omitempty"`
	FinishedAt string `json:"finishedAt,omitempty"`
	Status     string `json:"status,omitempty"`
	Discovered int    `json:"discovered"`
	Added      int    `json:"added"`
	Updated    int    `json:"updated"`
	Unchanged  int    `json:"unchanged"`
	Missing    int    `json:"missing"`
	Errors     int    `json:"errors"`
	Summary    string `json:"summary,omitempty"`
}

type ScanRun struct {
	ID         string `json:"id"`
	Status     string `json:"status"`
	StartedAt  string `json:"startedAt"`
	FinishedAt string `json:"finishedAt,omitempty"`
	Discovered int    `json:"discovered"`
	Added      int    `json:"added"`
	Updated    int    `json:"updated"`
	Unchanged  int    `json:"unchanged"`
	Missing    int    `json:"missing"`
	Errors     int    `json:"errors"`
	Summary    string `json:"summary"`
}

type ScanRunsResponse struct {
	Items []ScanRun `json:"items"`
}

type StartScanRequest struct {
	AllowMissing bool `json:"allowMissing"`
}

type HealthResponse struct {
	Status   string `json:"status"`
	Version  string `json:"version"`
	Database string `json:"database"`
	Library  string `json:"library"`
}

type VersionResponse struct {
	Version string `json:"version"`
}

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}
