package serverapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// APIError is a non-2xx server response.
type APIError struct {
	Status  int
	Code    string
	Message string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("server returned %d: %s", e.Status, e.Message)
}

// ClientInfo identifies a paired desktop client.
type ClientInfo struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Platform string `json:"platform"`
}

// Registration is the result of pairing.
type Registration struct {
	Client ClientInfo `json:"client"`
	Token  string     `json:"token"`
}

// Session is a server play session.
type Session struct {
	ID                       string `json:"id"`
	GameID                   string `json:"gameId"`
	Status                   string `json:"status"`
	StartedAt                string `json:"startedAt"`
	DurationSeconds          int    `json:"durationSeconds"`
	HeartbeatIntervalSeconds int    `json:"heartbeatIntervalSeconds"`
	TimeoutSeconds           int    `json:"timeoutSeconds"`
}

// Client is a small API client used by the desktop application.
type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

// New creates a client for the server base URL, for example
// http://pinas:8080.
func New(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: 15 * time.Second},
	}
}

// SetToken assigns the paired-client bearer token.
func (c *Client) SetToken(token string) {
	c.token = token
}

// Register exchanges a one-time pairing code for a client credential.
func (c *Client) Register(ctx context.Context, code, name, platform string) (Registration, error) {
	var result Registration
	err := c.do(ctx, http.MethodPost, "/api/v1/clients/register", map[string]string{
		"code":     code,
		"name":     name,
		"platform": platform,
	}, &result)
	return result, err
}

// StartSession asks the server to start a session for a game.
func (c *Client) StartSession(ctx context.Context, gameID string) (Session, error) {
	var result Session
	err := c.do(ctx, http.MethodPost, "/api/v1/sessions", map[string]string{"gameId": gameID}, &result)
	return result, err
}

// Heartbeat renews the session lease.
func (c *Client) Heartbeat(ctx context.Context, sessionID string) error {
	return c.do(ctx, http.MethodPost, "/api/v1/sessions/"+sessionID+"/heartbeat", nil, nil)
}

// Finish records the end of a session.
func (c *Client) Finish(ctx context.Context, sessionID, status string, exitCode *int) (Session, error) {
	var result Session
	err := c.do(ctx, http.MethodPost, "/api/v1/sessions/"+sessionID+"/finish", map[string]any{
		"status":   status,
		"exitCode": exitCode,
	}, &result)
	return result, err
}

func (c *Client) do(ctx context.Context, method, path string, body any, target any) error {
	if c.baseURL == "" {
		return fmt.Errorf("server URL is not configured")
	}
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode request: %w", err)
		}
		reader = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	response, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		apiErr := &APIError{Status: response.StatusCode, Code: "request_failed", Message: "request failed"}
		var envelope struct {
			Error struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		if decodeErr := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&envelope); decodeErr == nil {
			if envelope.Error.Code != "" {
				apiErr.Code = envelope.Error.Code
			}
			if envelope.Error.Message != "" {
				apiErr.Message = envelope.Error.Message
			}
		}
		return apiErr
	}
	if target == nil || response.StatusCode == http.StatusNoContent {
		return nil
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 4<<20)).Decode(target); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}
