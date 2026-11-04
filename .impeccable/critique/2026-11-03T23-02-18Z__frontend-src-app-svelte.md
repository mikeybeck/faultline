---
target: frontend
total_score: 27
max_score: 40
na_heuristics: 
p0_count: 0
p1_count: 2
target_identity: "file:/home/mikey/code/faultline/frontend/src/App.svelte"
target_fingerprint: "sha256:39fed742e568e310ffa74db2f9d3026dcac1a6919e0fc8fc438e51fa72d00368"
target_path: /home/mikey/code/faultline/frontend/src/App.svelte
timestamp: 2026-11-03T23-02-18Z
slug: frontend-src-app-svelte
---
# Critique: frontend Operate UI (post-fix)

Method: dual-agent (A: a1911fc5-ac6a-44f6-a2c8-1f22fc0269d6 · B: 17eb57d8-0de0-4a84-89fb-d06d3393c15e)

## Design Health Score
| # | Heuristic | Score | Key Issue |
|---|-----------|-------|-----------|
| 1 | Visibility of System Status | 3 | Pills/titles/toasts; quiet refresh |
| 2 | Match System / Real World | 3 | Mark clean / Mute need titles |
| 3 | User Control and Freedom | 3 | Esc/menus; no Mute undo |
| 4 | Consistency and Standards | 2 | #d44838 vs --accent; Views apply+delete |
| 5 | Error Prevention | 2 | Mute/Delete view lack confirm |
| 6 | Recognition Rather Than Recall | 3 | Unmute path is recall |
| 7 | Flexibility and Efficiency | 3 | Strong shortcuts; prompt() for views |
| 8 | Aesthetic and Minimalist Design | 3 | Groups land; Views+Mark clean still dense |
| 9 | Error Recovery | 2 | Raw String(e) toasts |
| 10 | Help and Documentation | 3 | ? modal; Settings dump |
| **Total** | | **27/40** | **Acceptable** (Good edge) |

## Design Specificity Verdict
**LLM:** Strong Fracture Console; post-fix deepens Operate craft. Two primaries (#d44838 / --accent).
**Deterministic scan:** 1× side-tab on .row.sel — selection FP. HTML-only path false-clean; CSS scan required. CDP N/A.
**Visual overlays:** N/A (WebKitGTK).

## Overall Impression
Toolbar/detail distill worked; score held at 27/40 with better heuristic balance. Highest leverage: Views destructive list + Mute undo/confirm.

## What's Working
1. Toolbar instrument groups.
2. Open primary + Copy/Snooze/Mute menus.
3. focus-visible / aria-pressed polish.

## Priority Issues
**[P1] Views menu apply+delete wall** — Apply-only; delete with confirm. → /impeccable distill
**[P1] Mute no guardrail** — Confirm/Undo toast. → /impeccable harden
**[P2] Settings/Welcome dump** — Progressive disclosure; Welcome primary. → /impeccable distill / onboard
**[P2] Primary token split** — Pin AA coral in DESIGN. → /impeccable document
**[P3] Save view window.prompt** — In-app field. → /impeccable polish

## Persona Red Flags
Alex: Views clutter; prompt save. Sam: color dots; menu keyboard incomplete; no focus trap. Mid-debug: Mute recovery; taxonomy of triage actions.

## Minor Observations
Detail still 5 chrome controls (over ≤4). Copy raw secrets title is good.

## Questions to Consider
1. Mark clean under Views so coral stays for Open?
2. Mute with undo toast?
3. Is #d44838 the new Fault Coral?
