---
target: site
total_score: 20
max_score: 32
na_heuristics: 7,10
p0_count: 0
p1_count: 2
target_identity: "file:/home/mikey/code/faultline/site/index.html"
target_fingerprint: "sha256:857f4a2166f3ec57437c31d3dc4ba8d19a26877e79e54051f0a85136066610cb"
target_path: /home/mikey/code/faultline/site/index.html
timestamp: 2026-11-03T22-35-33Z
slug: site-index-html
---
# Critique: site/index.html

Method: dual-agent (A: 5b318203-8f5f-45da-b8ec-7c27bcc5b7d9 · B: 618566d4-08df-4b8a-bb00-26b5cadfbba6)

## Design Health Score

| # | Heuristic | Score | Key Issue |
|---|-----------|-------|-----------|
| 1 | Visibility of System Status | 2 | Version on CTA; no active nav; mock status decorative |
| 2 | Match System / Real World | 3 | Strong dev vernacular; some insider jargon |
| 3 | User Control and Freedom | 3 | Clear section anchors; static page |
| 4 | Consistency and Standards | 3 | Download vs Install label drift |
| 5 | Error Prevention | 2 | Equal download cards invite wrong artifact |
| 6 | Recognition Rather Than Recall | 3 | Mac absence only in eyebrow |
| 7 | Flexibility and Efficiency | n/a | Persuade landing |
| 8 | Aesthetic and Minimalist Design | 2 | Green ambient + equal feature tiles dilute hierarchy |
| 9 | Error Recovery | 2 | Wrong binary → leave for GitHub |
| 10 | Help and Documentation | n/a | Persuade landing |
| **Total** | | **20/32** | **Acceptable** |

## Design Specificity Verdict

**LLM assessment**: Product-specific core (fault-bar, inbox mock, Open in editor) with category-generic shell (h1 as category phrase, 2×2 feature tiles, system type).

**Deterministic scan**: 25 findings on site/index.html — 23 low-contrast, 2 dark-glow. Most low-contrast pairs are false positives from gradient-on-body sampling against near-solid #101218. Keep: white on Fault Coral primary button (~3.4:1) and hover; dark-glow on Live Green status dots.

**Visual overlays**: No reliable user-visible overlay — browser automation not exposed.

## Overall Impression

The mock sells Faultline; conversion and brand hierarchy do not. Biggest opportunity: make Install one guided path and put Faultline at display level in the hero.

## What's Working
1. Product-as-hero mock (list/detail, fracture selection, mono evidence).
2. Honest Persuade surface — no fake social proof.
3. Token discipline matching Fracture Console.

## Priority Issues
**[P1] Install is a wall of equal artifacts** — Fix: recommend OS path; demote extension/TUI. Suggested: /impeccable distill
**[P1] Brand/category hierarchy inverted** — Faultline only in nav; h1 is generic. Suggested: /impeccable typeset
**[P2] Journey ends on raw binaries** — Add after-download beat. Suggested: /impeccable clarify
**[P2] Features are category furniture** — Asymmetric staged story. Suggested: /impeccable layout
**[P3] Live Green as page atmosphere** — Remove green wash. Suggested: /impeccable quieter

## Persona Red Flags
**Jordan**: unpacked extension / fingerprints jargon; opaque binary names; go install unexplained; Download vs Install.
**Riley**: extension card without browser steps; amd64-only; non-interactive mock looks operable.
**Casey**: CTAs not in thumb zone; small targets; long scroll of equal download cards.

## Minor Observations
No prefers-reduced-motion; weak focus-visible; gear emoji; sparse footer.

## Questions to Consider
1. If Faultline were removed from nav, would the first viewport still be uniquely this product?
2. What if Install answered only Linux vs Windows?
3. Should the last feeling be a filename or the file:line jump?
