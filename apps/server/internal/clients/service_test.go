package clients

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/lunararch/oryn/apps/server/internal/store"
)

func TestPairRegisterAuthenticateRevoke(t *testing.T) {
	f := newClientsFixture(t)
	ctx := context.Background()

	code, expires, err := f.service.CreatePairingCode(ctx)
	if err != nil {
		t.Fatalf("CreatePairingCode() error = %v", err)
	}
	if code == "" || expires <= time.Now().Unix() {
		t.Fatalf("code = %q, expires = %d", code, expires)
	}

	client, token, err := f.service.Register(ctx, code, "CachyOS Desktop", "linux")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if client.ID == "" || token == "" {
		t.Fatalf("client = %+v, token = %q", client, token)
	}
	if token[:5] != "oryn_" {
		t.Errorf("token prefix = %q", token[:5])
	}

	// The code is single-use.
	if _, _, err := f.service.Register(ctx, code, "Second", "windows"); !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("second Register() error = %v, want ErrInvalidCode", err)
	}

	authenticated, err := f.service.Authenticate(ctx, token)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if authenticated.ID != client.ID {
		t.Errorf("client ID = %q, want %q", authenticated.ID, client.ID)
	}

	list, err := f.service.List(ctx)
	if err != nil || len(list) != 1 {
		t.Fatalf("List() = %+v, error = %v", list, err)
	}

	if err := f.service.Revoke(ctx, client.ID); err != nil {
		t.Fatalf("Revoke() error = %v", err)
	}
	if _, err := f.service.Authenticate(ctx, token); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("Authenticate() after revoke error = %v, want ErrUnauthorized", err)
	}
}

func TestRegisterRejectsInvalidCode(t *testing.T) {
	f := newClientsFixture(t)
	if _, _, err := f.service.Register(context.Background(), "AAAA-BBBB", "Desktop", "linux"); !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("error = %v, want ErrInvalidCode", err)
	}
}

func TestPairingCodeExpires(t *testing.T) {
	f := newClientsFixture(t)
	code, _, err := f.service.CreatePairingCode(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	f.service.now = func() time.Time { return time.Now().Add(11 * time.Minute) }
	if _, _, err := f.service.Register(context.Background(), code, "Desktop", "linux"); !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("error = %v, want ErrInvalidCode", err)
	}
}

func TestTokenIsStoredAsHash(t *testing.T) {
	f := newClientsFixture(t)
	code, _, err := f.service.CreatePairingCode(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	_, token, err := f.service.Register(context.Background(), code, "Desktop", "linux")
	if err != nil {
		t.Fatal(err)
	}
	var stored string
	if err := f.store.DB.QueryRow(`SELECT token_hash FROM desktop_clients LIMIT 1`).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored == token {
		t.Fatal("plaintext token was stored")
	}
	if stored != hashToken(token) {
		t.Fatal("stored token hash does not match")
	}
}

func TestRegisterValidation(t *testing.T) {
	f := newClientsFixture(t)
	code, _, err := f.service.CreatePairingCode(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := f.service.Register(context.Background(), code, "", "linux"); !errors.Is(err, ErrValidation) {
		t.Fatalf("empty name error = %v", err)
	}
}

type clientsFixture struct {
	service *Service
	store   *store.Store
}

func newClientsFixture(t *testing.T) *clientsFixture {
	t.Helper()
	s, err := store.Open(filepath.Join(t.TempDir(), "oryn.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	if err := s.Migrate(context.Background()); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	return &clientsFixture{service: NewService(s, time.Now), store: s}
}
