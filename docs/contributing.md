# Contributing

Faultline is a Go + Wails desktop app (Linux/Windows), a TUI binary, a static marketing site, and an unpacked browser extension.

## Layout

| Path | Role |
|---|---|
| `main.go`, `app.go` | Desktop entry (Wails) and GUI bindings |
| `cmd/faultline` | TUI-only entry (no WebKit) |
| `frontend/` | Svelte UI embedded into the desktop binary |
| `extension/` | Manifest V3 extension (Chrome/Edge/Firefox) |
| `site/` | Static marketing site → [getfaultline.app](https://getfaultline.app) |
| `internal/engine` | Watch loop, triage, notifications |
| `internal/parser` | generic / JSON / Laravel / Apache parsers |
| `internal/source` | File follow + command runners |
| `internal/ingest` | Loopback HTTP server for the extension (`/ingest`, `/health`, `/hosts`, …) |
| `internal/persist` | SQLite inbox |
| `internal/sourcemap` | Remap generated JS frames |
| `internal/config` | YAML load / validate / ignore rules |
| `testdata/` | Sample logs and demo config |
| `PRODUCT.md`, `DESIGN.md` | Product positioning and visual system |

## Prerequisites

- Go 1.22+ (see `go.mod` for the pinned toolchain)
- Node.js 20+ (frontend)
- [Wails v2](https://wails.io) CLI for desktop builds
- Linux desktop: GTK3, WebKitGTK 4.1, libsoup 3 (see README)
- Windows cross-compile from Linux: `mingw-w64-gcc`
- `gh` to publish GitHub releases

## Develop

```bash
# Desktop (Linux uses webkit2_41 by default)
wails doctor
wails dev -tags webkit2_41

# Or rebuild
./build.sh                 # → build/bin/faultline
./build.sh tui             # TUI-only
./build.sh windows         # cross-compile .exe
./build.sh extension       # zip for Load unpacked / releases
```

Frontend lives under `frontend/`; Wails runs `npm install` / `npm run build` via `wails.json`.

## Test

```bash
go test ./internal/... ./cmd/...
go vet ./internal/... ./cmd/...
go fmt ./...
```

Prefer small table-driven tests next to the package you change (`internal/parser`, `internal/persist`, …).

## Release checklist

`build.sh release` builds Linux + Windows desktop binaries and the extension zip, pushes `HEAD`, and creates or updates a GitHub release. It does **not** update the marketing site.

1. Bump behavior and docs as needed. Version shown in the app comes from `git describe` / `VERSION=` via `-ldflags` into `internal/version`.
2. Update download links and version strings in:
   - `site/index.html`
   - `README.md` (Download + example `./build.sh release …` lines)
3. Publish binaries:

   ```bash
   NOTES='Short release notes.' ./build.sh release v0.1.15
   # optional NSIS installer:
   NOTES='…' ./build.sh release v0.1.15 -nsis
   ```

4. Deploy the site (Cloudflare Pages project `faultline` — **not** Git-connected):

   ```bash
   npx wrangler pages deploy site --project-name=faultline
   ```

5. Smoke-check [getfaultline.app](https://getfaultline.app) download URLs against the new release assets.

Local helpers `ship.sh` and `deploy.sh` (gitignored) commit, install to `~/bin/faultline`, and call `build.sh release` if you use them on a maintainer machine.

## Site only

```bash
npx wrangler pages deploy site --project-name=faultline
```

Wrangler’s local cache under `.wrangler/` is gitignored; do not commit it.

## Design / product docs

- User-facing behavior: root [README.md](../README.md)
- Config fields: [config.md](config.md)
- Common failures: [troubleshooting.md](troubleshooting.md)
- Positioning: [PRODUCT.md](../PRODUCT.md)
- Visual tokens and UI patterns: [DESIGN.md](../DESIGN.md)

Roadmap bullets in the README are ideas, not commitments.
