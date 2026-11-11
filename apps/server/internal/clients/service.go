package clients

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/lunararch/oryn/apps/server/internal/store"
	"github.com/lunararch/oryn/apps/server/internal/store/db"
)

var (
	// ErrUnauthorized is returned for an unknown or revoked credential.
	ErrUnauthorized = errors.New("unauthorized")
	// ErrInvalidCode is returned for an invalid, expired, or used pairing code.
	ErrInvalidCode = errors.New("invalid pairing code")
	// ErrValidation is returned for invalid client input.
	ErrValidation = errors.New("validation failed")
)

const (
	pairingCodeTTL = 10 * time.Minute
	codeAlphabet   = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
)

// Client is a paired desktop client.
type Client struct {
	ID           string
	Name         string
	Platform     string
	RegisteredAt int64
	LastSeenAt   int64
	RevokedAt    *int64
}

// Service manages pairing codes and desktop client credentials.
type Service struct {
	store *store.Store
	now   func() time.Time
}

// NewService builds a client service.
func NewService(s *store.Store, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{store: s, now: now}
}

// CreatePairingCode generates a one-time code valid for ten minutes.
func (s *Service) CreatePairingCode(ctx context.Context) (string, int64, error) {
	q := db.New(s.store.DB)
	now := s.now()
	if err := q.DeleteExpiredPairingCodes(ctx, now.Unix()); err != nil {
		return "", 0, fmt.Errorf("clean pairing codes: %w", err)
	}
	code, err := randomCode()
	if err != nil {
		return "", 0, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
	if err != nil {
		return "", 0, fmt.Errorf("hash pairing code: %w", err)
	}
	expires := now.Add(pairingCodeTTL)
	if err := q.CreatePairingCode(ctx, db.CreatePairingCodeParams{
		ID:        uuid.Must(uuid.NewV7()).String(),
		CodeHash:  string(hash),
		ExpiresAt: expires.Unix(),
		CreatedAt: now.Unix(),
	}); err != nil {
		return "", 0, fmt.Errorf("create pairing code: %w", err)
	}
	return code, expires.Unix(), nil
}

// Register exchanges a pairing code for a revocable client token.
func (s *Service) Register(ctx context.Context, code, name, platform string) (Client, string, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	name = strings.TrimSpace(name)
	platform = strings.TrimSpace(platform)
	if code == "" || name == "" || len(name) > 100 || platform == "" || len(platform) > 50 {
		return Client{}, "", ErrValidation
	}
	now := s.now()
	q := db.New(s.store.DB)
	codes, err := q.ListActivePairingCodes(ctx, now.Unix())
	if err != nil {
		return Client{}, "", fmt.Errorf("list pairing codes: %w", err)
	}
	var matched *db.PairingCode
	for i := range codes {
		if bcrypt.CompareHashAndPassword([]byte(codes[i].CodeHash), []byte(code)) == nil {
			matched = &codes[i]
			break
		}
	}
	if matched == nil {
		return Client{}, "", ErrInvalidCode
	}

	token, err := randomToken()
	if err != nil {
		return Client{}, "", err
	}
	client := Client{
		ID:           uuid.Must(uuid.NewV7()).String(),
		Name:         name,
		Platform:     platform,
		RegisteredAt: now.Unix(),
		LastSeenAt:   now.Unix(),
	}
	if err := q.CreateDesktopClient(ctx, db.CreateDesktopClientParams{
		ID:           client.ID,
		Name:         client.Name,
		Platform:     client.Platform,
		TokenHash:    hashToken(token),
		RegisteredAt: client.RegisteredAt,
		LastSeenAt:   client.LastSeenAt,
	}); err != nil {
		return Client{}, "", fmt.Errorf("create client: %w", err)
	}
	if err := q.MarkPairingCodeUsed(ctx, db.MarkPairingCodeUsedParams{
		UsedAt: sql.NullInt64{Int64: now.Unix(), Valid: true},
		ID:     matched.ID,
	}); err != nil {
		return Client{}, "", fmt.Errorf("consume pairing code: %w", err)
	}
	return client, token, nil
}

// Authenticate resolves a bearer token to a client.
func (s *Service) Authenticate(ctx context.Context, token string) (Client, error) {
	if !strings.HasPrefix(token, "oryn_") {
		return Client{}, ErrUnauthorized
	}
	q := db.New(s.store.DB)
	row, err := q.GetDesktopClientByTokenHash(ctx, hashToken(token))
	if err != nil {
		return Client{}, ErrUnauthorized
	}
	if row.RevokedAt.Valid {
		return Client{}, ErrUnauthorized
	}
	now := s.now().Unix()
	if err := q.TouchDesktopClient(ctx, db.TouchDesktopClientParams{
		LastSeenAt: now,
		ID:         row.ID,
	}); err != nil {
		return Client{}, fmt.Errorf("touch client: %w", err)
	}
	client := clientFromRow(row)
	client.LastSeenAt = now
	return client, nil
}

// List returns all paired clients.
func (s *Service) List(ctx context.Context) ([]Client, error) {
	rows, err := db.New(s.store.DB).ListDesktopClients(ctx)
	if err != nil {
		return nil, fmt.Errorf("list clients: %w", err)
	}
	items := make([]Client, 0, len(rows))
	for _, row := range rows {
		items = append(items, clientFromRow(row))
	}
	return items, nil
}

// Revoke invalidates a client credential.
func (s *Service) Revoke(ctx context.Context, id string) error {
	if err := db.New(s.store.DB).RevokeDesktopClient(ctx, db.RevokeDesktopClientParams{
		RevokedAt: sql.NullInt64{Int64: s.now().Unix(), Valid: true},
		ID:        id,
	}); err != nil {
		return fmt.Errorf("revoke client: %w", err)
	}
	return nil
}

func randomCode() (string, error) {
	raw := make([]byte, 8)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate pairing code: %w", err)
	}
	code := make([]byte, 8)
	for i, value := range raw {
		code[i] = codeAlphabet[int(value)%len(codeAlphabet)]
	}
	return string(code[:4]) + "-" + string(code[4:]), nil
}

func randomToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate client token: %w", err)
	}
	return "oryn_" + base64.RawURLEncoding.EncodeToString(raw), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func clientFromRow(row db.DesktopClient) Client {
	client := Client{
		ID:           row.ID,
		Name:         row.Name,
		Platform:     row.Platform,
		RegisteredAt: row.RegisteredAt,
		LastSeenAt:   row.LastSeenAt,
	}
	if row.RevokedAt.Valid {
		value := row.RevokedAt.Int64
		client.RevokedAt = &value
	}
	return client
}
