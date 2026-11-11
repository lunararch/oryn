# Oryn

Oryn is a self-hosted library and launcher for Ren'Py games and visual novels.

- The Go server on the Raspberry Pi indexes the game share, owns the SQLite database, and serves the API and web UI.
- The shared React frontend runs in the browser and, later, inside the Wails desktop client.
- Games are launched by the desktop client on the user's machine. The server never executes game content.
- The game share is treated as read-only. All Oryn data lives outside it.

The full product plan lives in `oryn_renpy-library-project-plan.md`. Project engineering rules are in `AGENTS.md` and `SKILL.md`.

## Status

Early implementation. The first milestone is the central library: scan a game directory, index Ren'Py games, and browse/manage them in the web UI.

## Repository layout

```text
apps/server      Go API server, scanner, SQLite store
apps/frontend    Shared React + TypeScript UI
apps/desktop     Wails desktop client (later milestone)
deployments      Docker Compose and Pi deployment files
docs             Architecture, API, development and deployment notes
```

## Development

Prerequisites: `mise` (Go, Node, sqlc, golangci-lint), npm, and a C toolchain only if a future dependency needs it. The server uses pure-Go SQLite.

```sh
make setup
make dev
```

`make dev` starts the API on `127.0.0.1:8080` and Vite on its development port. Vite proxies `/api` to the server.

Copy `.env.example` to `.env` for local reference and export the values you need. The server reads environment variables directly; it does not load `.env` files.

Generate TypeScript API types after changing Go request or response types:

```sh
go install github.com/gzuidhof/tygo@v0.2.21
make api-types
```

Run checks with:

```sh
make test
make lint
make build
```

## Configuration

| Variable | Default | Purpose |
|---|---|---|
| `ORYN_HTTP_ADDR` | `127.0.0.1:8080` | HTTP listen address |
| `ORYN_DATA_DIR` | `./data` | Application data outside the game share |
| `ORYN_DB_PATH` | `<data>/oryn.db` | SQLite database path |
| `ORYN_LIBRARY_ROOT` | unset | Read-only game library root |
| `ORYN_LIBRARY_SENTINEL` | unset | Existing path used to verify the library mount |
| `ORYN_WEB_DIR` | unset | Built frontend directory to serve |
| `ORYN_SCAN_INTERVAL` | `0` | Scheduled scan interval; `0` disables |
| `ORYN_SCAN_MAX_DEPTH` | `3` | Maximum directory depth |
| `ORYN_SCAN_WORKERS` | `2` | Bounded scanner concurrency |
| `ORYN_LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |

## Safety rules

- Never write to the game share.
- A missing or unmounted share is an infrastructure failure, not an empty library.
- Missing games are reconciled only after a clean, fully successful scan.
- Game scripts, executables, and installers are never executed by the server.
- The API is intended for a trusted LAN. Desktop clients will pair with revocable credentials in a later milestone.
