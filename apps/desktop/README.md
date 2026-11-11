# Oryn desktop

This module contains the security-critical desktop core:

- server-relative path resolution with traversal and symlink-escape rejection
- platform-specific launch candidate detection
- shell-free process launch and lifetime monitoring
- paired-client session API client
- launch + session tracking
- non-secret desktop configuration

The Wails 3 shell is not built yet. Wails v3 on Linux requires GTK4 and
`webkitgtk-6.0` development files, which are not installed in the development
environment (`webkit2gtk-4.1` is present instead). The core is deliberately
kept independent of Wails so it can be tested now and bound to a Wails service
layer once the required system packages are installed.

A diagnostic launcher is available:

```sh
cd apps/desktop
go run ./cmd/oryn-desktop-core \
  --server http://pinas:8080 \
  --token "$ORYN_CLIENT_TOKEN" \
  --root /mnt/games/visual-novels \
  --game "Example Game"
```

The token should come from the OS keyring in the final desktop client. The
diagnostic command accepts it via a flag or `ORYN_CLIENT_TOKEN`; do not commit
it or place it in the ordinary config file.

The shared React UI is the intended Wails frontend. When the Wails shell is
added, native launch actions must remain behind the typed desktop bridge; the
browser build must not import Wails APIs.

## Tests

```sh
cd apps/desktop
go test ./...
```

Tests launch only controlled executables such as `/bin/true`; they never launch
a real game.
