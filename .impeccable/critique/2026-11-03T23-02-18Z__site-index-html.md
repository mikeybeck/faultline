---
target: site
total_score: 22
max_score: 32
na_heuristics: 7,10
p0_count: 0
p1_count: 2
target_identity: "file:/home/mikey/code/faultline/site/index.html"
target_fingerprint: "sha256:7c13d40243f0bee9f07cee1f6a54721ef13840354d1d9a29d9810bcfcda6b1e0"
target_path: /home/mikey/code/faultline/site/index.html
timestamp: 2026-11-03T23-02-18Z
slug: site-index-html
---
# Critique: site/index.html (post-fix)

Method: dual-agent (A: 9d44c463-ac8c-4a08-b6ba-73c5a65f3a08 · B: c22d04ee-92fc-4276-8670-252dfe90679c)

## Design Health Score
| # | Heuristic | Score | Key Issue |
|---|-----------|-------|-----------|
| 1 | Visibility of System Status | 2 | No active nav; version only on hero CTA |
| 2 | Match System / Real World | 3 | Strong dev vernacular; some jargon |
| 3 | User Control and Freedom | 3 | Clear exits; details escapable |
| 4 | Consistency and Standards | 3 | Dual coral (#d44838 vs --accent); h1 tracking drift |
| 5 | Error Prevention | 3 | OS primary + Mac callout; thin Windows hint |
| 6 | Recognition Rather Than Recall | 3 | Numbered steps; after-download restates jump |
| 7 | Flexibility and Efficiency | n/a | Persuade |
| 8 | Aesthetic and Minimalist Design | 3 | Green wash gone; lede still dense |
| 9 | Error Recovery | 2 | Mac dead-end; no checksum path |
| 10 | Help and Documentation | n/a | Persuade |
| **Total** | | **22/32** | **Acceptable** (borderline Good) |

## Design Specificity Verdict
**LLM:** Moderately high — brand-first Faultline + mock triage grammar; still a familiar dark-tool landing shell.
**Deterministic scan:** 28 findings — ~20 FP (gradient-on-body); KEEP: white on #d44838 ~4.4:1 (improved), hover 3.4:1 on #e85d4c, dark-glow status dots, design-system-color for #d44838 outside DESIGN.md.
**Visual overlays:** Unavailable — no browser automation.

## Overall Impression
Post-fix Download IA and brand-first hero landed. Biggest remaining gap: tighten the lede and de-affordance the mock.

## What's Working
1. Brand + positioning split (h1 Faultline / lede strong).
2. Download: Linux/Windows primary; extras in details; after-download loop.
3. Watch→group→jump + coral ambient only.

## Priority Issues
**[P1] Lede claim wall** — One short sentence. → /impeccable clarify
**[P1] Mock false affordance** — De-button mock chrome. → /impeccable polish
**[P2] Mac dead-end** — Constructive next step. → /impeccable clarify
**[P2] Mobile hero buries Download** — Adapt stack. → /impeccable adapt
**[P3] h1 tracking vs DESIGN** — Align typeset. → /impeccable typeset

## Persona Red Flags
Jordan: dense lede; after-download thin. Riley: dead mock buttons; version hardcoding. Casey: tall mock before Download; small tap targets.

## Minor Observations
live.js inject still in site HTML — strip before production deploy. Leftover 3-col download-grid rule.

## Questions to Consider
1. Should the mock replace half the lede as the explanation?
2. Soft landing for Mac visitors?
3. OS-detect single primary download?
