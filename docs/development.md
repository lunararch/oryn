# Development

## Prerequisites

- Go 1.27+
- Node 24+ and npm
- `mise` (recommended for pinned tool versions)
- `sqlc` and `tygo` for code generation
- Docker for container checks
- Optional: `golangci-lint`

On CachyOS/Arch, Wails prerequisites will be needed for the desktop milestone (`webkit2gtk-4.1`, `gtk3`, `gcc`, `pkg-config`). They are not needed for the server/web milestone.

## First setup

```sh
mise install
cd apps/frontend && npm ci
cd ../..
```

Install code generators if they are not on `PATH`:

```sh
go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1
go install github.com/gzuidhof/tygo@v0.2.21
```

## Run locally

```sh
ORYN_LIBRARY_ROOT=/mnt/pinas/games_aapje make dev
```

This starts the API on `127.0.0.1:8080` and Vite on `5173`. Vite proxies `/api` to the server. The frontend is a single React application; do not create a second app for desktop.

Useful environment variables:

```sh
ORYN_DATA_DIR=./data
ORYN_DB_PATH=./data/oryn.db
ORYN_LIBRARY_ROOT=/mnt/pinas/games_aapje
ORYN_LIBRARY_SENTINEL=                # optional existing path
ORYN_SCAN_MAX_DEPTH=3
ORYN_SCAN_WORKERS=2
ORYN_SCAN_INTERVAL=0                  # 0 disables scheduled scans
ORYN_LOG_LEVEL=debug
```

The game share is read-only to Oryn. Never point `ORYN_DATA_DIR` or `ORYN_DB_PATH` inside the library root; configuration rejects that.

## Commands

```sh
make test          # Go + frontend tests
make lint          # go vet, golangci-lint if present, ESLint, tsc
make build         # frontend production build + server binary
make api-types     # regenerate TypeScript API types from Go DTOs
```

Run a single Go package:

```sh
cd apps/server
go test ./internal/scanner
```

Run a single frontend test:

```sh
cd apps/frontend
npm run test -- --run src/components/GameCard.test.tsx
```

Install the Playwright browser once and run the browser tests:

```sh
cd apps/frontend
npx playwright install chromium
npm run e2e -- --project=chromium
```

The E2E tests mock `/api/v1`; they exercise the shared UI without a backend.

## Code generation

Database access uses sqlc. After changing `query/*.sql` or migrations:

```sh
cd apps/server
sqlc generate
```

The generated code is committed under `internal/store/db`.

API types:

```sh
make api-types
```

This regenerates `apps/frontend/src/lib/api/generated.ts` from `apps/server/internal/api/types.go`. The generated file is committed so the frontend builds without the Go toolchain.

Migrations are embedded from `apps/server/migrations` and applied at startup. Add a new numbered migration; never edit an applied one.

## Test fixtures

Automated tests use synthetic Ren'Py-shaped directories created in temporary directories or committed under `apps/server/testdata`. Do not use the user's real game share as a fixture.

For manual checks, use a small subset such as `/mnt/pinas/games_aapje`. The full library contains 160 folders with many Windows-only builds, missing metadata, and archived artwork; do not run automated tests against it.

## Current server layout

```text
apps/server/
  cmd/oryn-server/      CLI entrypoint (serve, backup, version)
  internal/api/         Shared HTTP contract types
  internal/app/         Wiring, lifecycle, SPA serving
  internal/artwork/     Image validation and cache variants
  internal/config/      Environment validation
  internal/games/       Library queries and metadata overrides
  internal/httpapi/     Routes, handlers, DTO mapping
  internal/library/     Mount probe and library registration
  internal/scanner/     Detection, traversal, scan persistence
  internal/store/       SQLite, migrations, sqlc queries, backup
  migrations/           Versioned SQL
  query/                sqlc source queries
```

## Engineering rules

- Write a failing test before behavior changes.
- Do not write to the game share.
- A missing mount is not an empty library.
- Preserve metadata overrides across rescans.
- Use `os/exec` argument arrays for future desktop launching; never shell interpolation.
- Validate all client input on the server.
- Keep the API and database changes compatible and documented.
- Run `make test`, `make lint`, and `make build` before claiming completion.
