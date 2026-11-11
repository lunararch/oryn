# Raspberry Pi deployment and auto-update

This is the deployment used on `piNAS` (Raspberry Pi 5, ARM64).

## Current host layout

```text
/home/tim/Documents/projects/oryn   git checkout
/home/tim/shared/games              game library (read-only to Oryn)
/home/tim/oryn/data                 SQLite, artwork, backups
```

The server listens on host port `8082` because `soundtouch-web` already uses
`8080`. Compose maps `8082 -> 8080` inside the container.

## First install

```sh
cd ~/Documents/projects
git clone git@github.com:lunararch/oryn.git
cd oryn/deployments/docker
cp .env.example .env
```

Edit `.env` for the Pi:

```dotenv
ORYN_GAMES_PATH=/home/tim/shared/games
ORYN_DATA_PATH=/home/tim/oryn/data
ORYN_LIBRARY_SENTINEL=/games
ORYN_SCAN_INTERVAL=6h
ORYN_PORT=8082
ORYN_LOG_LEVEL=info
```

Then:

```sh
mkdir -p /home/tim/oryn/data
sudo docker compose --env-file .env up -d --build
curl -s http://127.0.0.1:8082/api/v1/health
```

The Pi user must be able to run `sudo -n docker`; the current `tim` account has
passwordless sudo. The container itself runs as UID/GID 1000 and the data
directory must be writable by that user.

## Auto-update

The updater is a small Go binary installed at `/usr/local/bin/oryn-updater`.
A systemd timer checks once per day at around 04:30, fetches the repository,
backs up the database, builds the new image, restarts the container, and checks
`/api/v1/health`. If the build or health check fails, it checks out the previous
commit, rebuilds, restarts, and reports the failed update.

Before updating, it runs:

```sh
docker compose ... exec -T oryn /app/oryn-server backup
```

The backup path is recorded in `/home/tim/oryn/update-state.json`.

### Release selection

- If a stable semantic-version tag such as `v0.1.0` is reachable from
  `origin/main`, the newest tag is used.
- If no stable tag exists, `origin/main` is used.
- Pre-release tags are ignored.
- Create a release with:

```sh
git tag -a v0.1.0 -m "Oryn v0.1.0"
git push origin v0.1.0
```

Once a stable tag exists, only tagged commits auto-deploy. This keeps `main`
free for work in progress.

### GitHub access

The Pi uses a read-only SSH deploy key at `~/.ssh/oryn_deploy`. Generate it on
the Pi:

```sh
ssh-keygen -t ed25519 -N "" -C "oryn-updater@pinas" -f ~/.ssh/oryn_deploy
cat ~/.ssh/oryn_deploy.pub
```

Add the public key in GitHub:

`Repository -> Settings -> Deploy keys -> Add deploy key` with **read access**
(do not enable write access).

Until the key is added, automatic updates fail during `git fetch`; the running
server is unaffected.

### Install the updater binary and units

Build the ARM64 binary on a development machine:

```sh
cd apps/server
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -o /tmp/oryn-updater ./cmd/oryn-updater
```

Copy it to the Pi and install:

```sh
scp /tmp/oryn-updater tim@pinas.local:/tmp/oryn-updater
ssh tim@pinas.local
sudo install -m 0755 /tmp/oryn-updater /usr/local/bin/oryn-updater
sudo install -m 0644 \
  ~/Documents/projects/oryn/deployments/pi/oryn-update.service \
  ~/Documents/projects/oryn/deployments/pi/oryn-update.timer \
  /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now oryn-update.timer
```

Run a check immediately:

```sh
sudo systemctl start oryn-update.service
journalctl -u oryn-update.service -n 100 --no-pager
cat ~/oryn/update-state.json
```

### Rollback limitations

The updater rolls back the code and image, not the SQLite schema or data. A
pre-update backup is always made unless `--skip-backup` is passed. Restore a
backup manually if a migration is ever incompatible with the previous binary:

```sh
sudo docker compose --env-file .env down
cp ~/oryn/data/backups/<backup>.db ~/oryn/data/oryn.db
sudo docker compose --env-file .env up -d
```

Migrations in this repository are currently additive, but future changes must
still be treated as forward-only in production.

## Security notes

- The Pi is on the trusted LAN. Do not expose port `8082` to the internet.
- The updater runs as `tim` and invokes `sudo -n docker`; it does not use a
  remote shell or download arbitrary commands.
- The deploy key is read-only and only authorizes this repository.
- Release tags should be created only after CI passes.
