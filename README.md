# Faultline

Local error inbox for any project that writes logs — or prints them to a terminal.

Faultline watches log files and command output, parses them into structured events, groups duplicates, and lets you jump to the offending `file:line` in your editor. Browser page errors arrive through an unpacked extension. The desktop app is the main interface; a terminal UI is still available.

```
┌─────────────────────────────────────────────────────────────┐
│ Faultline                         [app ✓] [errors …]   ⚙    │
│ [search]  all  error  warning     Recent ▾          Clear   │
├──────────────────────────┬──────────────────────────────────┤
│ TypeError ×3             │ TypeError                        │
│ payments.js:81           │ error · app · ×3                 │
│ 2s ago · app             │ payments.js:81                   │
│                          │ [Open in editor]  [Copy]         │
│ IntegrityError           │ Message + full stack             │
└──────────────────────────┴──────────────────────────────────┘
```

## Install

Desktop app (Go + Wails, needs Node.js 20+ to build the UI):

```bash
# Linux: GTK and WebKitGTK (Arch: gtk3 webkit2gtk-4.1)
# If pkg-config has webkit2gtk-4.1 rather than 4.0:
wails build -tags webkit2_41
```

The binary is written to `build/bin/faultline`.

TUI-only (no WebKit):

```bash
go install ./cmd/faultline
# or
go build -o faultline ./cmd/faultline
```

Requires Go 1.22+. On Linux, desktop notifications use `notify-send` (libnotify).

## Quick start (desktop)

1. Run `faultline` (or `wails dev` while developing).
2. **Open folder** — Faultline scans for `*.log` files (skipping `node_modules`, `vendor`, `.git`, …) and guesses the format (including JSON logs).
3. If nothing is found, **Add command** (`npm run dev`, `docker compose logs -f`, …) or **Add browser source**.
4. Or **Add log file** for any path, including a file that does not exist yet.
5. Choose your editor (installed editors are detected), then **Start watching**.
6. Load the browser extension (Welcome has the steps) so page errors show up in the inbox.

Settings are saved as `faultline.yaml` in the project folder. Recent projects are remembered.

Existing log lines are ingested on start, then new lines are followed. Pass `--tail` to skip what is already in the file.

```bash
faultline --config ./faultline.yaml
faultline --config ./faultline.yaml --tail
```

## Terminal UI

```bash
faultline --tui
faultline --tui --config ./faultline.yaml
faultline --tui --config ./faultline.yaml --tail
```

| Key | Action |
|---|---|
| `↑` / `k` | Move up |
| `↓` / `j` | Move down |
| `Enter` | Toggle detail view |
| `o` | Open in editor |
| `d` | Dismiss until it happens again |
| `c` | Mark inbox clean |
| `R` | Replay log files |
| `/` | Filter |
| `s` | Toggle sort by frequency |
| `q` | Quit |

Desktop shortcuts (inbox): `/` search, `j`/`k` or arrows move, `Enter` focus detail, `o` open, `d` dismiss, `c` mark clean, `s` sort, `?` help.

## Browser errors

The extension captures page errors without adding a script tag to your app. Faultline listens on `127.0.0.1:9477` while a project is being watched.

Load it unpacked (the folder is `extension/` in this repo):

- **Chrome / Edge / Chromium:** `chrome://extensions` → Developer mode → Load unpacked → select `extension/`
- **Firefox 128+:** `about:debugging#/runtime/this-firefox` → Load Temporary Add-on → select `extension/manifest.json`

It injects into local origins (`localhost`, `127.0.0.1`, `*.test`, `*.local`, `*.ddev.site`, `*.lndo.site`). Custom names such as `localphishingbox.com` are not included by default — add them under **Settings → Extra browser hosts** (Faultline serves them at `http://127.0.0.1:9477/hosts`) or the extension’s **Options**. Uncaught errors, unhandled promise rejections, and errors a framework swallows then prints with `console.error` (Vue `v-on` handlers, for example) are sent to Faultline; if Faultline is not running the requests fail silently.

A `browser` source in `faultline.yaml` is optional (custom listen address, or a browser-only project with no log files):

```yaml
sources:
  - name: browser
    type: browser
    path: 127.0.0.1:9477
browser:
  extraHosts:
    - localphishingbox.com
```

## What it does

| Capability | Details |
|---|---|
| Sources | Log files (`generic`, `json`, Laravel, Apache), command stdout/stderr, and browser errors via the extension |
| Structured events | type, message, file, line, severity, stack, counts |
| Dedup | fingerprints group repeats (`×N`, first/last seen) |
| Desktop UI | source health, split inbox + detail, filter, severity, sort, clickable stacks |
| Notifications | desktop alert on **new** fingerprints only |
| Editor | opens VS Code / Cursor / PhpStorm / custom command at file:line |
| Browser | unpacked extension posts `window.onerror` / `unhandledrejection` / `console.error(Error)` to `127.0.0.1:9477` |

## Log formats

### Generic (default)

Error-like lines (`error`, `fatal`, `panic`, `exception`, `traceback`, `warn`, …) with common stacks from Node, Python, Go, Java, PHP, and Ruby. Info/debug noise is skipped. `file:line` is extracted when present. JSON lines (Pino, Winston, Zap) in a mixed stream are parsed too.

### JSON

Newline-delimited JSON logs. Levels may be strings (`error`, `warn`) or Pino numbers (`50`, `40`). Nested `err` / `error` objects supply type, message, and stack.

### Command

`type: command` runs a shell command in the project folder and parses stdout/stderr with the generic parser:

```yaml
sources:
  - name: vite
    type: command
    path: npm run dev
```

### Laravel

Parses Monolog-style lines such as:

```text
[2026-07-15 14:31:01] local.ERROR: TypeError: ...
#0 /var/www/app/Http/Controllers/PaymentController.php(81): ...
```

Uses the first non-`vendor` frame when choosing file/line.

### Apache error log

Parses common bracketed formats and PHP messages. Access logs are not watched.

Format is sniffed when you add a file; you can override it in the UI.

## Editor setup

```yaml
editor:
  command: code        # VS Code
  # command: cursor
  # command: phpstorm
  # command: "nvim +{line} {file}"
```

Templates may include `{file}` and `{line}`.

## Demo with sample logs

```bash
mkdir -p /tmp/faultline-demo
cp testdata/generic_sample.log /tmp/faultline-demo/app.log
cp testdata/laravel_sample.log /tmp/faultline-demo/laravel.log

cat > /tmp/faultline-demo/faultline.yaml <<'EOF'
sources:
  - name: app
    type: generic
    path: /tmp/faultline-demo/app.log
  - name: laravel
    type: laravel
    path: /tmp/faultline-demo/laravel.log
notifications:
  enabled: false
editor:
  command: code
EOF

# Desktop:
faultline --config /tmp/faultline-demo/faultline.yaml

# Terminal:
faultline --tui --config /tmp/faultline-demo/faultline.yaml
```

Then append another line to see live updates:

```bash
echo 'ERROR TypeError: demo in /tmp/App.js:1' >> /tmp/faultline-demo/app.log
```

## Development

```bash
go test ./internal/... ./cmd/...
go vet ./internal/... ./cmd/...
go fmt ./...
wails doctor
wails dev -tags webkit2_41
wails build -tags webkit2_41
```

## Roadmap ideas

- Clickable notification → editor (platform-specific)
- Sound themes and saved views
