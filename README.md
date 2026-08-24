# Faultline

Local error awareness for PHP / Laravel apps.

Faultline watches log files, parses them into structured events, groups duplicates, shows a TUI inbox, and can notify your desktop and open the right file in your editor.

```
┌──────────────────────────────────────────┐
│ Faultline                                │
│ Watching                                 │
│ ✓ laravel  storage/logs/laravel.log      │
│ ✓ apache   /var/log/apache2/error.log    │
├──────────────────────────────────────────┤
│ New Errors (2)                           │
│                                          │
│ ● TypeError ×3                           │
│   PaymentController.php:81               │
│   2 seconds ago · laravel                │
│                                          │
│ ● SQLSTATE[23000]                        │
│   User.php:41                            │
│   8 minutes ago · laravel                │
├──────────────────────────────────────────┤
│ ↑/↓ · Enter · o open · c clear · q quit  │
└──────────────────────────────────────────┘
```

## Install

```bash
cd faultline
go install ./cmd/faultline
```

Or build a local binary:

```bash
go build -o faultline ./cmd/faultline
```

Requires Go 1.22+. On Linux, desktop notifications use `notify-send` (libnotify).

## Quick start

1. Copy the example config into your app project:

```bash
cp faultline.example.yaml /path/to/your-app/faultline.yaml
```

2. Edit paths:

```yaml
sources:
  - name: laravel
    type: laravel
    path: storage/logs/laravel.log

  - name: apache
    type: apache
    path: /var/log/apache2/error.log

notifications:
  enabled: true
  sound: false

editor:
  command: code   # or phpstorm, or "nvim +{line} {file}"
```

3. Run from that project directory:

```bash
faultline
# or
faultline --config ./faultline.yaml
```

By default Faultline starts at EOF (only new lines). To ingest existing content:

```bash
faultline --from-start
```

## What it does

| Capability | Details |
|---|---|
| Sources | Laravel `*.log`, Apache `error.log` |
| Structured events | type, message, file, line, severity, stack, counts |
| Dedup | fingerprints group repeats (`Occurred ×N`, first/last seen) |
| TUI | source health, error inbox, detail view, filter, sort by frequency |
| Notifications | desktop alert on **new** fingerprints only |
| Editor | `o` opens VS Code / PhpStorm / custom command at file:line |

## Shortcuts

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

## Supported formats

### Laravel

Parses Monolog-style lines such as:

```text
[2026-07-15 14:31:01] local.ERROR: TypeError: ...
#0 /var/www/app/Http/Controllers/PaymentController.php(81): ...
```

Uses the first non-`vendor` frame when choosing file/line.

### Apache error log

Parses common bracketed formats and PHP messages:

```text
[Wed Jul 15 14:31:01.123456 2026] [php:error] [pid 1234] [client 127.0.0.1:54321] PHP Fatal error: ... in /var/www/app/Http/Controllers/PaymentController.php on line 81
```

Access logs are not watched in this MVP.

## Editor setup

```yaml
editor:
  command: code        # VS Code / Cursor code CLI
  # command: phpstorm  # JetBrains toolbox launcher
  # command: "nvim +{line} {file}"
```

Templates may include `{file}` and `{line}`.

## Demo with sample logs

```bash
mkdir -p /tmp/faultline-demo
cp testdata/laravel_sample.log /tmp/faultline-demo/laravel.log
cp testdata/apache_sample.log /tmp/faultline-demo/apache.log

cat > /tmp/faultline-demo/faultline.yaml <<'EOF'
sources:
  - name: laravel
    type: laravel
    path: /tmp/faultline-demo/laravel.log
  - name: apache
    type: apache
    path: /tmp/faultline-demo/apache.log
notifications:
  enabled: false
editor:
  command: code
EOF

faultline --config /tmp/faultline-demo/faultline.yaml --from-start
```

Then append another line to see live updates:

```bash
echo '[2026-07-15 15:00:00] local.ERROR: RuntimeException: demo in /tmp/App.php on line 1' >> /tmp/faultline-demo/laravel.log
echo >> /tmp/faultline-demo/laravel.log
```

## Development

```bash
go test ./...
go vet ./...
go fmt ./...
```

## Roadmap ideas

- Docker Compose / `npm run dev` command sources
- Clickable notification → editor (platform-specific)
- Optional browser detail view
- Sound themes and severity filters in config
