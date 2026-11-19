# Troubleshooting

## Browser status stays “waiting for the extension”

1. Start watching a project in Faultline so ingest listens on `127.0.0.1:9477`.
2. Load the unpacked extension (Welcome steps, or unzip `faultline-extension.zip` from a release).
3. Open a page on a matching origin (`localhost`, `127.0.0.1`, `*.test`, `*.local`, `*.ddev.site`, `*.lndo.site`, or a host listed under **Settings → Extra browser hosts** / extension Options).
4. Confirm `http://127.0.0.1:9477/health` returns `{"ok":true}` from the machine.
5. Firefox: temporary add-ons are cleared when the browser restarts — reload via `about:debugging` each session.

Custom hostnames (for example `myapp.internal`) are **not** injected until added to `browser.extraHosts` or extension Options. Faultline serves the list at `http://127.0.0.1:9477/hosts`.

If Faultline is not running, the extension fails silently.

## No errors from the page, but DevTools shows them

The extension captures:

- Uncaught errors and `unhandledrejection`
- Some framework-swallowed errors that end up as `console.error(Error)` (for example Vue `v-on` handlers)
- Failed `fetch` / XHR (except HTTP 404), connection failures
- HTTP 200 JSON bodies with `success: false` and an error message

It does **not** turn every `console.log` / `console.warn` into an inbox row. Access logs and successful responses are ignored.

## Secrets in the inbox

Failed browser calls include the request body and up to ~8KB of the response. Login forms, tokens, and cookies in those bodies can appear in the detail pane and in local SQLite (`inbox.db`).

- Data stays on the machine; nothing is uploaded by Faultline.
- Dismiss, mute, or mark clean as needed; wiping `inbox.db` clears stored groups for all projects.
- Prefer not to trigger auth failures against real credentials while testing capture.

## Desktop build fails on Linux (WebKit)

Install WebKitGTK 4.1 and friends, then build with the `webkit2_41` tag (default in `./build.sh` on Linux):

```bash
# Ubuntu/Debian
sudo apt install libgtk-3-dev libwebkit2gtk-4.1-dev libsoup-3.0-dev pkg-config

wails doctor
./build.sh
```

macOS has no prebuilt desktop app yet; use the TUI (`go install ./cmd/faultline` or `./build.sh tui`).

## TUI: “no faultline.yaml found”

Copy the example and point at it, or run the desktop app to create config via Open folder:

```bash
cp faultline.example.yaml faultline.yaml
faultline --tui --config ./faultline.yaml
```

## Inbox empty after restart / after Mark clean

**Mark clean** stores resume offsets in `marks.yaml` so already-cleared log lines are skipped on the next launch. Use **Replay logs** to forget offsets and re-read files (browser events are kept).

`--tail` also skips existing file content for that run.

## Notifications never appear

- Check `notifications.enabled` in `faultline.yaml` / Settings.
- Alerts fire only for **new** fingerprints, not repeats.
- Linux needs `notify-send` (libnotify). macOS TUI/desktop-notify path expects `terminal-notifier` when used. Windows uses balloon notifications from the desktop build.

## Editor does not open at the right line

Confirm `editor.command` (`code`, `cursor`, `phpstorm`, or a template with `{file}` and `{line}`). Sourcemaps remap generated JS when a sibling `.map`, inline `data:` map, or loopback HTTP map is available; without a map, Faultline opens the generated `file:line`.

## Self-diagnostics

Faultline appends its own failures to `faultline.log` beside app config and can show them in the inbox as source `faultline`. See [config.md — Data on disk](config.md#data-on-disk).

## Site downloads point at the wrong version

The Pages project is **not** Git-synced. After a release, bump `site/index.html` (and README) then:

```bash
npx wrangler pages deploy site --project-name=faultline
```
