---
name: Faultline
description: Dark local error inbox — fracture console for developers
colors:
  fault-coral: "#d44838"
  hot-ash: "#ff8a72"
  fault-coral-dim: "rgba(212, 72, 56, 0.16)"
  primary-hover: "#e85d4c"
  pit-charcoal: "#101218"
  panel-slate: "#171b24"
  panel-raised: "#1e2430"
  hairline: "#2a3140"
  hairline-hover: "#3d4658"
  text-bright: "#f4f6fb"
  text-muted: "#c5cddc"
  text-faint: "#9aa3b8"
  live-green: "#3dd68c"
  wait-amber: "#f0b429"
  err-coral: "#ff7a6e"
  crit-rose: "#ff4d6d"
  on-accent: "#ffffff"
  stack-well: "#161a22"
typography:
  display:
    fontFamily: "Segoe UI, system-ui, sans-serif"
    fontSize: "clamp(36px, 5vw, 52px)"
    fontWeight: 650
    lineHeight: 1.05
    letterSpacing: "-0.02em"
  headline:
    fontFamily: "Segoe UI, system-ui, sans-serif"
    fontSize: "28px"
    fontWeight: 650
    lineHeight: 1.2
    letterSpacing: "-0.02em"
  title:
    fontFamily: "Segoe UI, system-ui, sans-serif"
    fontSize: "22px"
    fontWeight: 650
    lineHeight: 1.25
    letterSpacing: "0em"
  body:
    fontFamily: "Segoe UI, system-ui, sans-serif"
    fontSize: "16px"
    fontWeight: 400
    lineHeight: 1.5
    letterSpacing: "0em"
  body-app:
    fontFamily: "Segoe UI, system-ui, sans-serif"
    fontSize: "14.5px"
    fontWeight: 400
    lineHeight: 1.45
    letterSpacing: "0em"
  label:
    fontFamily: "Segoe UI, system-ui, sans-serif"
    fontSize: "12px"
    fontWeight: 400
    lineHeight: 1.4
    letterSpacing: "0.06em"
  mono:
    fontFamily: "ui-monospace, SF Mono, Cascadia Code, Menlo, Consolas, monospace"
    fontSize: "13px"
    fontWeight: 400
    lineHeight: 1.6
    letterSpacing: "0em"
rounded:
  sm: "6px"
  md: "8px"
  control: "10px"
  panel: "12px"
  card: "14px"
  modal: "16px"
  welcome: "18px"
  pill: "999px"
spacing:
  xs: "6px"
  sm: "8px"
  md: "12px"
  lg: "16px"
  xl: "18px"
  "2xl": "24px"
  "3xl": "28px"
  "4xl": "40px"
components:
  button-primary:
    backgroundColor: "{colors.fault-coral}"
    textColor: "{colors.on-accent}"
    rounded: "{rounded.control}"
    padding: "8px 14px"
  button-primary-hover:
    backgroundColor: "{colors.primary-hover}"
    textColor: "{colors.on-accent}"
    rounded: "{rounded.control}"
    padding: "8px 14px"
  button-secondary:
    backgroundColor: "{colors.panel-slate}"
    textColor: "{colors.text-bright}"
    rounded: "{rounded.control}"
    padding: "8px 12px"
  button-secondary-hover:
    backgroundColor: "{colors.panel-raised}"
    textColor: "{colors.text-bright}"
    rounded: "{rounded.control}"
    padding: "8px 12px"
  chip-on:
    backgroundColor: "{colors.fault-coral-dim}"
    textColor: "{colors.text-bright}"
    rounded: "{rounded.pill}"
    padding: "5px 10px"
  chip-off:
    backgroundColor: "transparent"
    textColor: "{colors.text-bright}"
    rounded: "{rounded.pill}"
    padding: "5px 10px"
  input-search:
    backgroundColor: "{colors.panel-slate}"
    textColor: "{colors.text-bright}"
    rounded: "{rounded.control}"
    padding: "8px 12px"
  inbox-row-selected:
    backgroundColor: "{colors.fault-coral-dim}"
    textColor: "{colors.text-bright}"
    padding: "10px 16px"
  card-panel:
    backgroundColor: "{colors.panel-slate}"
    textColor: "{colors.text-bright}"
    rounded: "{rounded.welcome}"
    padding: "28px"
---

# Design System: Faultline

## Overview

**Creative North Star: "The Fracture Console"**

Faultline’s UI is a dark instrument panel for local error triage. Surfaces stay in charcoal pits and slate panels; the coral fracture — accent, selection inset, and the thin top fault-bar — marks where something broke. Density is high and utilitarian: inbox rows, chips, and mono stacks outrank decorative chrome. The same token set binds the desktop app, marketing site, and extension options so the product feels like one console whether you are downloading or triaging.

Personality is precise and instrument-like: calm fields, sparse accent, glowing health dots only when status matters. Reject pastel SaaS dashboards, purple glow stacks, and light “friendly” marketing chrome. Depth comes from tonal steps and hairline borders; one soft ambient shadow lifts cards, modals, and the marketing mock — not every control.

**Key Characteristics:**
- Shared dark token set across site, desktop, and extension
- Coral fracture used sparingly for CTA, selection, and the fault-bar
- System sans for UI; mono reserved for paths, stacks, and code
- Soft radii (8–18px) with pill chips; hairline borders as structure
- Tonal elevation first; ambient shadow only on lifted panels

## Colors

A near-monochrome charcoal field with one coral fracture accent and traffic-light status colors for source health and severity.

### Primary
- **Fault Coral** (`#d44838`): Primary CTAs, selected chip/pill borders, inbox selection inset, brand fracture. Tuned for ~4.5:1 with white labels. Keep rare so it stays loud.
- **Hot Ash** (`#ff8a72`): Softer accent for links, counts (`×N`), clickable locations, and label accents.
- **Fault Coral Dim** (`rgba(212, 72, 56, 0.16)`): Selected row/chip wash and callout backgrounds.
- **Primary Hover** (`#e85d4c`): Primary button hover (lighter coral).

### Neutral
- **Pit Charcoal** (`#101218`): App and page canvas; field wells inside elevated panels.
- **Panel Slate** (`#171b24`): Elevated surfaces — cards, feature tiles, secondary buttons, search.
- **Panel Raised** (`#1e2430`): Hover elevation and menu popovers.
- **Hairline** (`#2a3140`): Structural borders and dividers.
- **Hairline Hover** (`#3d4658`): Control border on hover.
- **Text Bright** (`#f4f6fb`): Primary copy and brand wordmark.
- **Text Muted** (`#c5cddc`): Supporting copy, meta, ledes.
- **Text Faint** (`#9aa3b8`): Tertiary meta, idle dots, footnotes.
- **Stack Well** (`#161a22`): Message and stack code blocks in the detail pane.

### Status (not decorative accents)
- **Live Green** (`#3dd68c`): Healthy source pill dots (with soft glow).
- **Wait Amber** (`#f0b429`): Waiting sources and warning severity; also ends the fault-bar gradient.
- **Err Coral** (`#ff7a6e`): Error severity labels.
- **Crit Rose** (`#ff4d6d`): Critical severity and failing source dots.

**The Fracture Rule.** Fault Coral and Hot Ash together should occupy a small fraction of any screen. Status greens/ambers are for health and severity only — never as brand decoration.

## Typography

**Display Font:** Segoe UI (with system-ui, sans-serif)
**Body Font:** Segoe UI (with system-ui, sans-serif)
**Label/Mono Font:** ui-monospace / SF Mono / Cascadia Code / Menlo / Consolas

**Character:** Native system UI type — no custom webfonts. Hierarchy comes from weight (~650), tracking, and size, not ornament. Mono is the signature for anything that is a path, stack, or config.

### Hierarchy
- **Display** (650, `clamp(36px, 5vw, 52px)`, 1.05, −0.02em): Marketing hero only.
- **Headline** (650, 28px, −0.02em): Section titles (Features, Download) and welcome card title.
- **Title** (650, 22px): Detail pane event type; brand wordmark in chrome is 17px with +0.04em tracking.
- **Body** (400, 16px site / 14.5px app, 1.5 / 1.45): Prose and inbox density.
- **Label** (400, 11–13px, +0.06em, often uppercase): Severity tags, block headings, eyebrows.
- **Mono** (400, 11–14px): Stacks, `file:line`, install snippets, path inputs.

**The Mono Is Evidence Rule.** If the string is a filesystem path, stack frame, YAML, or shell command, it is mono. Do not style decorative UI labels in mono.

## Layout

Marketing content sits in a **1080px** max column with **24px** side padding (**16px** under 860px). The hero is a two-column grid (~0.9 / 1.1) that collapses to one column below 860px. Feature and download grids are 2- and 3-column respectively, also collapsing to one.

The desktop app is a full-viewport flex column: fault-bar → chrome → toolbar → split body. The inbox body is **~38% / 62%** list/detail (`minmax(280px, 38%) 1fr`), stacking to one column under 840px. Fixed list row height is **76px**. Rhythm uses tight gaps (**6–12px**) in toolbars and **14–28px** in marketing and welcome cards. Density favors scanability over whitespace theater.

**The Split Console Rule.** Inbox triage is always list + detail (or stacked equivalent). Do not replace the split with card grids of errors.

## Elevation & Depth

Primarily **tonal layering**: canvas → elevated → raised, separated by hairlines. Ambient shadow (`0 18px 50px rgba(0, 0, 0, 0.35)`) appears on lifted panels only — welcome/settings cards, modals, toasts, menus, and the marketing app mock. Top chrome uses translucent pit charcoal with **12px** backdrop blur. Selected inbox rows use a coral wash plus a **3px** inset accent bar — selection is a fracture mark, not a drop shadow.

### Shadow Vocabulary
- **Lifted panel** (`box-shadow: 0 18px 50px rgba(0, 0, 0, 0.35)`): Cards, modals, menus, mock window, toasts.
- **Live glow** (`box-shadow: 0 0 8px` on Live Green): Healthy source status dots only.

**The Flat-By-Default Rule.** Controls and list rows stay flat. Shadows are for floating overlays and hero mocks, never for every button.

## Shapes

Corners are gently rounded, not sharp and not bubbly: **6–10px** on controls and fields, **12–14px** on feature/mock panels, **16–18px** on modals and welcome cards. Chips and source pills are full pills (**999px**). Borders are **1px** hairlines; selected filters gain a coral border. The signature silhouette is the **3px fault-bar** gradient across the top edge (transparent → Fault Coral → Hot Ash → Wait Amber → transparent).

**The Soft Instrument Rule.** Prefer 8–12px radii for interactive chrome. Reserve larger radii (14–18px) for primary floating containers.

## Components

Precise and instrument-like: quiet secondary chrome, coral only when the action is primary or the row is selected.

### Buttons
- **Shape:** Soft control radius (10px); small variant 8px.
- **Primary:** Fault Coral fill, white text, weight 600, padding ~8×12–14px; hover Primary Hover.
- **Secondary:** Panel Slate fill, hairline border; hover raises to Panel Raised / Hairline Hover.
- **Ghost:** Transparent background, same border language.
- **Danger:** Err Coral text on secondary chrome.
- **Motion:** Border/background ~0.15s; light transform ~0.1s on app buttons.

### Chips
- **Style:** Pill outline on hairline; 12px type; transparent off-state.
- **On:** Fault Coral Dim fill + Fault Coral border + bright text — used for severity/source filters.

### Cards / Containers
- **Corner Style:** 12px feature tiles; 14px mock; 18px welcome card; 16px modal.
- **Background:** Panel Slate (wells use Pit Charcoal).
- **Shadow Strategy:** Lifted panel shadow on welcome/modal/mock; feature tiles are bordered only.
- **Border:** 1px Hairline; download cards hover to Fault Coral border.
- **Internal Padding:** ~16–28px depending on scale.

### Inputs / Fields
- **Style:** Panel Slate or Pit Charcoal well, hairline border, 8–10px radius, ~8–11px padding.
- **Focus:** Border shifts to Fault Coral (search and fields); no glow ring.
- **Mono fields:** Paths and YAML-ish textareas use the mono stack at 13px.

### Navigation
- **Marketing:** Brand wordmark (bright, 650, +0.04em) left; muted nav links; primary Download button.
- **App chrome:** Brand + version, source pills with status dots, settings icon button; translucent blurred top bar under the fault-bar.

### Inbox row (signature)
- Fixed 76px row; title + severity; meta `file:line` and age/source.
- Selected: Fault Coral Dim wash + 3px inset Fault Coral bar.
- Count badge in Hot Ash tabular nums.

### Fault-bar (signature)
- 3px full-bleed gradient strip at the top of site and app — the literal fracture line.

## Do's and Don'ts

### Do:
- **Do** keep Pit Charcoal → Panel Slate → Panel Raised as the only surface ladder.
- **Do** use Fault Coral for primary action, selection fracture, and the fault-bar — then stop.
- **Do** put paths, stacks, and commands in mono.
- **Do** preserve the list/detail inbox split and 76px row rhythm in the app.
- **Do** reuse the shared CSS variables rather than inventing parallel palettes per surface.

### Don't:
- **Don't** introduce light themes, purple accents, or multi-stop decorative glows.
- **Don't** put drop shadows on every control or list row.
- **Don't** use Live Green / Wait Amber as brand decoration outside status and severity.
- **Don't** swap in display webfonts that fight the system Segoe/UI stack without an explicit rebrand.
- **Don't** invent testimonials or product claims in marketing chrome — visual system stays honest to PRODUCT.md evidence.
