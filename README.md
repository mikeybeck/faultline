# Faultline

Local error inbox for any project that writes log files.

Faultline watches logs, parses them into structured events, groups duplicates, and lets you jump to the offending `file:line` in your editor. The desktop app is the main interface; a terminal UI is still available.

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
2. **Open folder** — Faultline scans for `*.log` files (skipping `node_modules`, `vendor`, `.git`, …) and guesses the format.
3. Or **Add log file** for any path, including a file that does not exist yet.
4. Choose your editor, then **Start watching**.

Settings are saved as `faultline.yaml` in the project folder. The last project is remembered.

By default only **new** lines are ingested. Turn on **Read existing log content** (or pass `--from-start`) to load what is already in the file.

```bash
faultline --config ./faultline.yaml --from-start
```

## Terminal UI

```bash
faultline --tui
faultline --tui --config ./faultline.yaml --from-start
```

| Key | Action |
|---|---|
| `↑` / `k` | Move up |
| `↓` / `j` | Move down |
| `Enter` | Toggle detail view |
| `o` | Open in editor |
| `c` | Clear all events |
| `/` | Filter |
| `s` | Toggle sort by frequency |
| `q` | Quit |

Desktop shortcuts (inbox): `/` search, `o` open, `c` clear.

## What it does

| Capability | Details |
|---|---|
| Sources | Any `*.log` file (`generic`), plus Laravel Monolog and Apache error logs |
| Structured events | type, message, file, line, severity, stack, counts |
| Dedup | fingerprints group repeats (`×N`, first/last seen) |
| Desktop UI | source health, split inbox + detail, filter, severity, sort |
| Notifications | desktop alert on **new** fingerprints only |
| Editor | opens VS Code / Cursor / PhpStorm / custom command at file:line |

## Log formats

### Generic (default)

Error-like lines (`error`, `fatal`, `panic`, `exception`, `traceback`, `warn`, …) with common stacks from Node, Python, Go, Java, PHP, and Ruby. Info/debug noise is skipped. `file:line` is extracted when present.

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
faultline --config /tmp/faultline-demo/faultline.yaml --from-start

# Terminal:
faultline --tui --config /tmp/faultline-demo/faultline.yaml --from-start
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

- Docker Compose / `npm run dev` command sources
- Clickable notification → editor (platform-specific)
- Sound themes and saved views
