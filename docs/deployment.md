# Deployment

Oryn's server targets a Raspberry Pi 5 running Raspberry Pi OS (64-bit) or another Linux ARM64 distribution. The game share must be mounted read-only on the host and passed into the container.

## Layout

```text
/opt/oryn/
  compose.yaml
  .env
  data/
    oryn.db
    artwork/
    backups/
    logs/

/srv/games/            # read-only SMB mount on the host
```

Application data must live outside the game share. The container runs as UID/GID `1000`; the data directory must be writable by that user.

## Build and start

From `deployments/docker`:

```sh
cp .env.example .env
# edit ORYN_GAMES_PATH and optional sentinel
mkdir -p data
sudo chown -R 1000:1000 data
docker compose --env-file .env up -d --build
```

The image is multi-stage: Node builds the web UI, Go builds a static server, and Alpine runs both as a non-root user. For ARM64, build on the Pi or use CI/buildx with `--platform linux/arm64`.

`compose.yaml` uses a bind mount with `create_host_path: false` for `/games`, so Compose fails if the host path is missing instead of silently creating an empty directory.

## Mount safety

- Mount the SMB share on the host and make sure it is available before starting Oryn.
- Set `ORYN_LIBRARY_SENTINEL` to an existing file or directory inside the share. If the sentinel is missing, scans fail without changing records.
- Oryn refuses to scan when the root is unreadable.
- If a readable root discovers zero games while the database already has games, the scan fails and preserves records.
- Missing-game reconciliation only runs after a clean scan. A mass-missing threshold blocks accidental mass deletion; the UI exposes an explicit override.

Check health:

```sh
curl -s http://127.0.0.1:8080/api/v1/health
```

`database` should be `ok`; `library` should be `available`. The container healthcheck treats process/database failure as unhealthy; library availability is reported in the body.

## Configuration

| Variable | Default | Notes |
|---|---|---|
| `ORYN_HTTP_ADDR` | `:8080` in the container | Keep bound to the LAN address |
| `ORYN_DATA_DIR` | `/data` | SQLite, artwork, backups (container path) |
| `ORYN_DATA_PATH` | `./data` | Host path bind-mounted to `/data` |
| `ORYN_WEB_DIR` | `/app/web` | Built frontend |
| `ORYN_LIBRARY_ROOT` | `/games` | Container path for the read-only share |
| `ORYN_LIBRARY_SENTINEL` | unset | Existing path inside the share |
| `ORYN_SCAN_INTERVAL` | `6h` | `0` disables scheduled scans |
| `ORYN_SCAN_MAX_DEPTH` | `3` | Maximum discovery depth |
| `ORYN_SCAN_WORKERS` | `2` | Keep low on a Pi |
| `ORYN_LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |

Do not expose the port to the public internet. There is no user authentication yet; Oryn is for a trusted LAN.

## Backups

The server can write a consistent SQLite backup without stopping:

```sh
docker compose exec oryn /app/oryn-server backup
```

The default destination is `/data/backups/oryn-YYYYMMDD-HHMMSS.db`. You can pass an explicit path:

```sh
docker compose exec oryn /app/oryn-server backup /data/backups/manual.db
```

The command refuses to overwrite an existing file. Copy backups off the Pi regularly.

### Restore

1. Stop the container: `docker compose down`.
2. Move the current database aside.
3. Copy the chosen backup to `/data/oryn.db`.
4. Start the container and check `/api/v1/health`.

Do not copy a live database file while it is being written; use the backup command.

## Upgrades

1. Pull the new source or image.
2. Take a backup.
3. Rebuild/start the container. Embedded migrations run at startup.
4. Check health and run one scan.
5. Verify metadata overrides and artwork are intact.

Because the game share is read-only and all Oryn state lives in `/data`, the container can be recreated without touching game files.

## Troubleshooting

- **Health says library unavailable**: the host mount is missing or the sentinel does not exist. Fix the mount; do not rescan repeatedly.
- **Scan fails with "no games discovered"**: the root is readable but empty or the mount is wrong. Existing records were preserved.
- **Scan is partial**: some directories could not be read. Missing-game reconciliation was skipped. Check permissions and logs, then scan again.
- **Artwork missing**: many games keep art inside `.rpa` archives or have no loose cover. Upload a cover from the game details page.
- **Container cannot write `/data`**: ensure the host directory is owned by UID/GID 1000 or adjust the Compose `user`.
- **Port conflict**: change `ORYN_PORT` in `.env`.
