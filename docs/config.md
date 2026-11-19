# Configuration reference

Project settings live in YAML next to the code you are watching. The desktop UI writes the same file when you change Settings.

## File names

Faultline looks for, in order:

1. `faultline.yaml`
2. `faultline.yml`
3. `.faultline.yaml`
4. `.faultline.yml`

Desktop: last opened project is remembered under the user config dir (see [Data on disk](#data-on-disk)). TUI without `--config` searches the current working directory.

Start from the checked-in template:

```bash
cp faultline.example.yaml faultline.yaml
```

At least one `sources` entry is required.

## Top-level shape

```yaml
sources: []          # required
notifications: {}
editor: {}
inbox: {}            # optional
browser: {}          # optional
```

## `sources`

Each source has:

| Field | Required | Meaning |
|---|---|---|
| `name` | no | Label in the UI; default `{type}-{n}` |
| `type` | yes* | `generic`, `json`, `laravel`, `apache`, `command`, or `browser` (*empty becomes `generic`) |
| `path` | yes† | Log file path, shell command, or ingest listen address (†optional for `browser`; default `127.0.0.1:9477`) |
| `parser` | no | For `command` only: `generic`, `json`, `laravel`, or `apache` (default generic) |

Examples:

```yaml
sources:
  - name: app
    type: generic
    path: logs/app.log

  - name: api
    type: json
    path: logs/api.log

  - name: vite
    type: command
    path: npm run dev
    parser: generic

  - name: browser
    type: browser
    path: 127.0.0.1:9477
```

Notes:

- File sources are tailed; missing files are watched until they appear.
- `command` runs in the project directory; stdout and stderr are parsed together.
- A `browser` source is optional. While a project is watched, Faultline still serves ingest on `127.0.0.1:9477` unless you override the address.
- Open-folder scan looks for `*.log` (depth ≤ 5, ≤ 40 files) and skips `node_modules`, `vendor`, `.git`, `dist`, `.next`, `target`, virtualenvs, and other dot-dirs. Format is sniffed from content (JSON / Laravel / Apache / generic).

## `editor`

| Field | Default | Meaning |
|---|---|---|
| `command` | `code` | `code`, `cursor`, `phpstorm`, an executable path, or a template with `{file}` and `{line}` |

```yaml
editor:
  command: "nvim +{line} {file}"
```

## `notifications`

| Field | Default | Meaning |
|---|---|---|
| `enabled` | false when omitted as empty object; UI defaults on for new projects | Desktop alert on **new** fingerprints only |
| `sound` | `false` | Platform sound when supported |

```yaml
notifications:
  enabled: true
  sound: false
```

## `inbox`

| Field | Default | Meaning |
|---|---|---|
| `followLatest` | `false` | Always select the newest error as it arrives |
| `clearOnCommit` | `false` | Hide current errors when `.git/HEAD` changes |
| `ignore` | `[]` | Checked-in drop rules (see below) |
| `views` | `[]` | Named filter presets |

### Ignore rules

Every set field on a rule must match (AND). Any matching rule drops the event (OR). Empty rules are invalid.

| Field | Match |
|---|---|
| `type` | Case-insensitive equality with event type |
| `message` | Case-insensitive substring of the message |
| `path` | Substring, or glob (`*`, `?`, `[]`, `**/…`, `…/**`) against `file` |
| `source` | Exact source name |
| `regex` | Case-insensitive regex against message, type, or file |

```yaml
inbox:
  followLatest: false
  clearOnCommit: true
  ignore:
    - type: DeprecationWarning
    - message: ECONNRESET
    - path: "**/node_modules/**"
    - source: vite
    - regex: "heartbeat"
```

Changing ignore rules in Settings requires **Save and restart**.

### Saved views

| Field | Required | Meaning |
|---|---|---|
| `name` | yes | Label in the Views menu |
| `filter` | no | Search string |
| `severity` | no | Severity filter |
| `source` | no | Source name filter |
| `sort` | no | e.g. `recent` (frequency sort is toggled in the UI) |

```yaml
inbox:
  views:
    - name: Browser errors
      source: browser
      severity: error
      sort: recent
```

## `browser`

| Field | Meaning |
|---|---|
| `extraHosts` | Hostnames (no scheme) the extension should also inject into, beyond the built-in local patterns |

```yaml
browser:
  extraHosts:
    - localphishingbox.com
```

Faultline exposes the list at `http://127.0.0.1:9477/hosts` while watching. You can also edit hosts in the extension Options page.

## CLI flags

| Flag | Default | Meaning |
|---|---|---|
| `--config path` | last project (desktop) or search cwd (TUI) | Config file |
| `--tui` | off | Terminal UI (desktop binary only; `cmd/faultline` is always TUI) |
| `--from-start` | on | Ingest existing log content, then follow |
| `--tail` | off | Follow new lines only (overrides `--from-start`) |

```bash
faultline --config ./faultline.yaml
faultline --config ./faultline.yaml --tail
faultline --tui --config ./faultline.yaml
```

## Data on disk

Under the OS user config directory (`~/.config/faultline` on Linux, `%AppData%\faultline` on Windows, `~/Library/Application Support/faultline` on macOS):

| Path | Purpose |
|---|---|
| `inbox.db` | SQLite groups, mutes, snoozes, dismissals (keyed by project) |
| `state.yaml` | Last / recent projects (up to 8) |
| `marks.yaml` | Log resume offsets after **Mark clean** |
| `faultline.log` | Faultline’s own errors (also shown as source `faultline`) |
| `extension/` | Unpacked extension copied from the desktop app for Load unpacked |

Project config (`faultline.yaml`) stays in the project folder and is safe to commit when you want shared ignore rules and sources. Local secrets in failed browser request bodies are stored in `inbox.db` on that machine only — see [Troubleshooting](troubleshooting.md#secrets-in-the-inbox).
