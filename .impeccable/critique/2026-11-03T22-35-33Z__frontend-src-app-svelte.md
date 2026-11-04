---
target: frontend
total_score: 27
max_score: 40
na_heuristics: 
p0_count: 0
p1_count: 3
target_identity: "file:/home/mikey/code/faultline/frontend/src/App.svelte"
target_fingerprint: "sha256:39fed742e568e310ffa74db2f9d3026dcac1a6919e0fc8fc438e51fa72d00368"
target_path: /home/mikey/code/faultline/frontend/src/App.svelte
timestamp: 2026-11-03T22-35-33Z
slug: frontend-src-app-svelte
---
# Critique: frontend Operate UI

Method: dual-agent (A: 769130f3-788d-4971-9fde-c42ce0865ef0 · B: c5dffa15-a10e-4cb4-b3c9-543bd03805e5)

## Design Health Score

| # | Heuristic | Score | Key Issue |
|---|-----------|-------|-----------|
| 1 | Visibility of System Status | 3 | Pills/toasts strong; disabled Open lacks why |
| 2 | Match System / Real World | 3 | Dev vernacular; Mark clean needs tooltips |
| 3 | User Control and Freedom | 2 | No undo for Dismiss/Mute |
| 4 | Consistency and Standards | 3 | Follow as chip and Settings; Views peer controls |
| 5 | Error Prevention | 2 | Mute/Dismiss matching lack confirm |
| 6 | Recognition Rather Than Recall | 3 | Shortcuts behind ?; unmute only in Settings |
| 7 | Flexibility and Efficiency | 4 | Strong keyboard triage loop |
| 8 | Aesthetic and Minimalist Design | 2 | Toolbar + 8 detail buttons too peer-weighted |
| 9 | Error Recovery | 2 | Raw String(e) toasts |
| 10 | Help and Documentation | 3 | ? modal; weak Mark clean vs Dismiss vs Mute |
| **Total** | | **27/40** | **Acceptable** (bordering Good) |

## Design Specificity Verdict

**LLM assessment**: Strong Fracture Console identity (fault-bar, selection fracture, mono evidence, source pills). Specificity dilutes where every power feature is equally visible.

**Deterministic scan**: 1× side-tab at frontend/src/style.css:178 (.row.sel inset accent) — contextual FP for selected list row, not decorative card. CDP/overlay N/A (WebKitGTK). CLI undercounts (Svelte/CSS regex only).

**Visual overlays**: Not applicable — desktop WebKitGTK; no browser automation.

## Overall Impression

Excellent instrument identity and keyboard loop; chrome density fights the single job (select → open). Biggest opportunity: distill toolbar and detail actions around Open in editor.

## What's Working
1. Signature Operate grammar.
2. Keyboard triage (/ j k o d z c s ?).
3. State-aware empty copy.

## Priority Issues
**[P1] Toolbar wall of peer actions** — Chunk/overflow. Suggested: /impeccable distill
**[P1] Detail strip 8 buttons** — Primary Open; Copy/Snooze menus. Suggested: /impeccable distill
**[P1] a11y focus + color-as-status** — focus-visible, aria-pressed, dialogs. Suggested: /impeccable audit
**[P2] Irreversible triage without undo** — Toast Undo. Suggested: /impeccable harden
**[P2] Follow latest fights selection** — Session mode vs persisted. Suggested: /impeccable clarify

## Persona Red Flags
**Alex**: Enter focuses detail not open; prompt() for views; Mute click-only.
**Sam**: Weak focus-visible; color-only health dots; no focus trap on dialogs.
**Dev mid-debug**: Open disabled vs loc ambiguity; secrets in Raw with no caution; toolbar noise.

## Minor Observations
No sources non-button pill; welcome four add actions need a recommended primary.

## Questions to Consider
1. Why do three Copy variants share rank with Open in editor?
2. Should Mark clean be a footer ritual not a red toolbar peer?
3. Should Follow latest be session-only?
