# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Users

Developers (solo or small teams) mid-local work — watching log files, a terminal, or a local browser — who need to find and open the real `file:line` for a repeating error without leaving their machine for a cloud dashboard.

## Product Purpose

Faultline is a local error inbox for any project that writes logs or prints them to a terminal. It watches log files and command output, parses them into structured events, groups duplicates, and opens the offending location in the developer's editor. Browser page errors can arrive through an unpacked extension. Success is: the right bug is one row, and jumping to `file:line` is immediate.

## Positioning

Local-first error triage — not a hosted APM or Sentry replacement. It watches existing logs and command streams (and optional browser errors without a script tag in the app), keeps data on the machine, and is built around editor jump-to-source rather than cloud aggregation.

## Operating Context

- Desktop app (Go + Wails, Linux and Windows) is the main interface; a TUI remains available (`faultline --tui`).
- Open a project folder; Faultline scans for `*.log` files, or the user adds log files, shell commands (`npm run dev`, `docker compose logs -f`, …), or a browser source.
- Settings live in project `faultline.yaml`; recent projects are remembered; SQLite inbox persists under the OS config directory.
- Browser extension posts local-origin page errors to `127.0.0.1:9477` while a project is watched.
- Marketing site is static HTML in `site/`, hosted at https://getfaultline.app (Cloudflare Pages).
- Source and releases: https://github.com/mikeybeck/faultline.

## Capabilities and Constraints

- Sources: log files (generic, JSON, Laravel, Apache), command stdout/stderr, browser errors via extension.
- Dedup by fingerprint; dismiss / mute / snooze / mark clean; ignore rules; saved views; sourcemap remapping; desktop notifications on new fingerprints; editor open at file:line.
- Local-only ingest; request/response bodies from failed fetches may contain secrets and appear in the inbox.
- Desktop targets Linux and Windows; TUI builds without WebKit; extension is unpacked (Chrome/Edge/Firefox).
- Undecided: roadmap ideas in the README (sound themes, test-runner parsers, git blame) are not product commitments.

## Brand Commitments

- Product name: **Faultline**.
- Public site: **getfaultline.app**.
- GitHub: **mikeybeck/faultline**.
- Positioning language already in product copy: "local error inbox"; desktop app · Linux and Windows.

## Evidence on Hand

- README.md product and capability documentation.
- Marketing page and mock inbox UI in `site/`.
- Sample logs and demo config under `testdata/`.
- No customer testimonials, case studies, pricing, or third-party benchmarks in-repo — future work must not fabricate them.

## Product Principles

1. Stay local — errors never need a cloud account to be useful.
2. One row per bug — fingerprint and triage so noise collapses to actionable groups.
3. Jump to source — the payoff is opening the real `file:line` in the editor the developer already uses.
4. Fit existing workflows — watch logs and commands they already run; optional browser capture without instrumenting the app.
5. Honest product surface — claim only what ships; do not invent social proof or SaaS features.
