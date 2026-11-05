---
name: Tollseal
description: Every AI provider behind one gateway key, and every request inspected, numbered and sealed. Printed as a security document.
colors:
  banknote-green: "#0d5c48"
  banknote-green-deep: "#0a4a3a"
  banknote-green-tint: "#d6e7df"
  banknote-paper-on-green: "#f2f8f5"
  banknote-green-night: "#4cc9a4"
  banknote-green-night-hover: "#6ad6b6"
  banknote-ink-on-green-night: "#052119"
  teal-iris: "#0f7a76"
  teal-iris-tint: "#cfe6e2"
  teal-iris-night: "#4fc7bd"
  intaglio-indigo: "#262f5e"
  intaglio-indigo-night: "#8fa4e6"
  foil-teal: "#2aa39a"
  foil-blue: "#4166c8"
  foil-gold: "#c9a43a"
  foil-gold-night: "#e2c15f"
  prism-cap: "#1c2550"
  intaglio-red: "#a3271f"
  intaglio-red-stamp: "#861f19"
  intaglio-red-night: "#f0736a"
  ochre: "#8a5f12"
  ochre-night: "#e2b24f"
  indigo-info: "#3250a8"
  security-paper: "#edf1ee"
  security-paper-shade: "#e4ebe7"
  note-paper: "#f8faf9"
  float-paper: "#fcfdfc"
  paper-press: "#dfe7e3"
  paper-press-active: "#d9e3de"
  intaglio-ink: "#141a33"
  ink-secondary: "#3a4363"
  ink-muted: "#4e5776"
  hairline: "rgba(20, 26, 51, 0.12)"
  hairline-strong: "rgba(20, 26, 51, 0.22)"
  guilloche-line: "rgba(15, 122, 118, 0.24)"
  guilloche-line-soft: "rgba(38, 47, 94, 0.09)"
  night-paper: "#12102a"
  night-paper-shade: "#1d1a3d"
  night-note: "#191736"
  night-float: "#1e1b40"
  night-press: "#24204a"
  night-ink: "#ecebf8"
  night-ink-muted: "#a9a5c9"
  night-hairline: "rgba(236, 235, 248, 0.09)"
  night-hairline-strong: "rgba(236, 235, 248, 0.18)"
  night-guilloche-line: "rgba(79, 199, 189, 0.2)"
  terminal-ink: "#0f1530"
  terminal-text: "#e3e8f5"
  terminal-dim: "#97a1c0"
  chart-indigo: "#232b5c"
  chart-green: "#1d7a5f"
  chart-teal: "#2aa39a"
  chart-ochre: "#c49a2c"
  chart-slate: "#8a93a8"
typography:
  display:
    fontFamily: "Archivo Variable, Archivo, ui-sans-serif, system-ui, sans-serif"
    fontSize: "clamp(2.6rem, 6.2vw, 5.25rem)"
    fontWeight: 640
    lineHeight: 0.96
    letterSpacing: "-0.02em"
    fontVariation: "'wdth' 118"
  headline:
    fontFamily: "Archivo Variable, Archivo, ui-sans-serif, system-ui, sans-serif"
    fontSize: "clamp(2rem, 4vw, 3.25rem)"
    fontWeight: 640
    lineHeight: 1.02
    letterSpacing: "-0.02em"
    fontVariation: "'wdth' 118"
  numeral:
    fontFamily: "Archivo Variable, Archivo, ui-sans-serif, system-ui, sans-serif"
    fontSize: "1.75rem"
    fontWeight: 640
    lineHeight: 1
    letterSpacing: "-0.02em"
    fontFeature: "'tnum' 1"
    fontVariation: "'wdth' 118"
  title:
    fontFamily: "Geist Sans, ui-sans-serif, system-ui, sans-serif"
    fontSize: "0.9375rem"
    fontWeight: 600
    lineHeight: 1.375
    letterSpacing: "-0.01em"
  body-lead:
    fontFamily: "Geist Sans, ui-sans-serif, system-ui, sans-serif"
    fontSize: "1.0625rem"
    fontWeight: 400
    lineHeight: 1.625
  body:
    fontFamily: "Geist Sans, ui-sans-serif, system-ui, sans-serif"
    fontSize: "0.875rem"
    fontWeight: 400
    lineHeight: 1.625
  label:
    fontFamily: "Geist Mono, ui-monospace, SFMono-Regular, monospace"
    fontSize: "0.6875rem"
    fontWeight: 400
    lineHeight: 1.4
    letterSpacing: "0.12em"
  microprint:
    fontFamily: "Geist Mono, ui-monospace, SFMono-Regular, monospace"
    fontSize: "0.5625rem"
    fontWeight: 400
    lineHeight: 1.4
    letterSpacing: "0.18em"
  code:
    fontFamily: "Geist Mono, ui-monospace, SFMono-Regular, monospace"
    fontSize: "0.8125rem"
    fontWeight: 400
    lineHeight: 1.625
rounded:
  stamp: "4px"
  sm: "5px"
  md: "7.5px"
  field: "10px"
  xl: "14px"
  panel: "18px"
  note: "20px"
  pill: "9999px"
spacing:
  hairline: "1px"
  xs: "8px"
  sm: "12px"
  md: "16px"
  lg: "20px"
  xl: "24px"
  2xl: "40px"
  section: "96px"
  section-lg: "128px"
  container: "1240px"
components:
  button-primary:
    backgroundColor: "{colors.banknote-green}"
    textColor: "{colors.banknote-paper-on-green}"
    typography: "{typography.title}"
    rounded: "{rounded.pill}"
    padding: "0 16px"
    height: "36px"
  button-primary-hover:
    backgroundColor: "{colors.banknote-green-deep}"
    textColor: "{colors.banknote-paper-on-green}"
  button-primary-hero:
    backgroundColor: "{colors.banknote-green}"
    textColor: "{colors.banknote-paper-on-green}"
    rounded: "{rounded.pill}"
    padding: "0 28px"
    height: "48px"
  button-outline:
    backgroundColor: "{colors.note-paper}"
    textColor: "{colors.intaglio-ink}"
    rounded: "{rounded.pill}"
    padding: "0 16px"
    height: "36px"
  button-outline-hover:
    backgroundColor: "{colors.paper-press}"
    textColor: "{colors.intaglio-ink}"
  button-ghost:
    textColor: "{colors.intaglio-ink}"
    rounded: "{rounded.pill}"
    padding: "0 16px"
    height: "36px"
  button-destructive:
    textColor: "{colors.intaglio-red}"
    rounded: "{rounded.pill}"
    padding: "0 16px"
    height: "36px"
  input:
    backgroundColor: "{colors.note-paper}"
    textColor: "{colors.intaglio-ink}"
    rounded: "{rounded.field}"
    padding: "4px 12px"
    height: "36px"
  badge:
    backgroundColor: "{colors.security-paper-shade}"
    textColor: "{colors.intaglio-ink}"
    rounded: "{rounded.pill}"
    padding: "2px 10px"
    height: "24px"
  stamp-specimen:
    textColor: "{colors.intaglio-red-stamp}"
    typography: "{typography.label}"
    rounded: "{rounded.stamp}"
    padding: "2px 8px"
  stamp-sample-figures:
    textColor: "{colors.ink-muted}"
    typography: "{typography.label}"
    rounded: "{rounded.stamp}"
    padding: "2px 8px"
  panel:
    backgroundColor: "{colors.note-paper}"
    textColor: "{colors.intaglio-ink}"
    rounded: "{rounded.panel}"
    padding: "20px"
  note:
    backgroundColor: "{colors.note-paper}"
    textColor: "{colors.intaglio-ink}"
    rounded: "{rounded.note}"
    padding: "80px 48px 40px"
  ledger-cell:
    backgroundColor: "{colors.note-paper}"
    textColor: "{colors.intaglio-ink}"
    typography: "{typography.numeral}"
    padding: "16px"
  code-terminal:
    backgroundColor: "{colors.terminal-ink}"
    textColor: "{colors.terminal-text}"
    typography: "{typography.code}"
    rounded: "{rounded.panel}"
    padding: "20px"
---

# Design System: Tollseal

## Overview

**Creative North Star: "The Sealed Note"**

Tollseal is printed, not rendered. Every surface is a security document: cool security paper, fine guilloche linework cut by a computed lathe, deep intaglio indigo ink, serial numbers in the corners, microprint along the edge. A request is a document of value: it passes the toll, is inspected check by check, and leaves sealed and numbered. The marketing site persuades with one large note (the hero) and a sequence of ruled registers and ledgers; the console operates on the same paper, quieter, its figures set like denominations. Dark mode is night intaglio: indigo-black paper and luminous teal linework, the same hues lifted, not a separate theme.

Density is calm and ruled. Information lives in hairline-divided registers and gap-px ledgers inside one bordered panel, rarely in grids of separate cards. Colour sits on data and on the single primary action; everything else is ink on paper. The iris foil (teal to blue to gold) is the one luxurious material and is held back for the gateway prism and the seal. The world refuses the dark dev-tool hero with glow, the grid of feature cards, and the warm ivory editorial world it replaced.

Motion engraves and stamps once, then rests. The page is complete without JavaScript; motion only reveals what the markup already holds.

**Key Characteristics:**
- Security printing: guilloche rosettes and bands, microprint strips, serial numbers, SPECIMEN overprints.
- Banknote green for the one primary action and the seal mark; intaglio red for blocked and danger.
- Iris foil only on the gateway prism and seal.
- Archivo set wide (118% width, weight 640) for denominations: headlines and figures. Geist for the interface, Geist Mono for serials, codes and check labels.
- Ruled registers and gap-px ledgers instead of card grids.
- Light "security paper" and dark "night intaglio", sharing hues.

## Colors

Ink on cool paper, one green note, a foil kept for the seal, and a red that only ever means stop.

### Primary
- **Banknote Green** (light `banknote-green`, hover `banknote-green-deep`; night `banknote-green-night`, hover `banknote-green-night-hover`): the one primary action per view (Get started, the default button), the seal mark (`LogoMark` prints in the primary), bars in sample charts, and the stress clause of a display headline ("One gateway key.", "checked and sealed."). Text on it is `banknote-paper-on-green` by day and `banknote-ink-on-green-night` by night.
- **Banknote Green Tint** (`banknote-green-tint`, night `#123a31`): quiet primary fill behind selected or confirmed states.

### Secondary
- **Teal Iris** (`teal-iris`, night `teal-iris-night`; tint `teal-iris-tint`): the working accent. Focus rings, caret, text selection (28% mix), section and stat icons, step numerals in the developers band, the setup progress bar, the active-nav dot. It is never the primary action.
- **Intaglio Indigo** (`intaglio-indigo`, night `intaglio-indigo-night`): the engraved ink of the brand. First chart series, the "AI" accent token, and the lifted ink of night mode.

### Tertiary
- **Iris Foil** (`foil-teal` to `foil-blue` to `foil-gold`, set as `linear-gradient(115deg, ...)` in the `--foil` token; night stops `#4fc7bd`, `#84a2f0`, `foil-gold-night`): the gateway prism's plates only. The prism cap is `prism-cap` with a gold edge, and the seal on it prints in `foil-gold-night`.

### Status
Pages write Tailwind's `emerald`, `green`, `teal` / `amber`, `yellow`, `orange` / `rose`, `red` / `blue`, `sky`, `cyan` / `violet`, `purple`, `indigo` families; `globals.css` remaps them once onto five ramps (`--sage-*` banknote green, `--honey-*` ochre, `--brick-*` intaglio red, `--dusk-*` indigo, `--lav-*` teal iris). Use shade 600/700 for text on paper and 400 on night paper.
- **Intaglio Red** (`intaglio-red`, night `intaglio-red-night`): destructive actions, blocked requests, error figures, risk icons in the problem register, and the SPECIMEN overprint (`intaglio-red-stamp` text on a half-strength border).
- **Ochre** (`ochre`, night `ochre-night`): warnings and near-limit figures.
- **Indigo Info** (`indigo-info`, night `#84a2f0`): informational states.
- **Success** shares the sage ramp (`#1d6b50` by day, `banknote-green-night` by night).

### Neutral
- **Security Paper** (`security-paper`; night `night-paper`): the page ground and sidebar.
- **Paper Shade** (`security-paper-shade`; night `night-paper-shade`): secondary and muted fills, table header bands, empty bars.
- **Note Paper** (`note-paper`; night `night-note`): cards, panels and the hero note.
- **Float Paper** (`float-paper`; night `night-float`): popovers, menus, the command palette.
- **Paper Press** (`paper-press`, active nav `paper-press-active`; night `night-press`): hover and pressed fills.
- **Intaglio Ink** (`intaglio-ink`; night `night-ink`): text. Secondary text `ink-secondary`; muted text `ink-muted` / `night-ink-muted`, both above 4.5:1 on their papers.
- **Hairline** (`hairline`, `hairline-strong`; night variants): every border. Inputs use ink at 20% (night 15%).
- **Guilloche Line** (`guilloche-line`, `guilloche-line-soft`; night `night-guilloche-line`): the only colours guilloche is drawn in.
- **Terminal Ink** (`terminal-ink`, text `terminal-text`, dim `terminal-dim`; night ground `#0c0a1f`): the code panel, which stays dark in both modes.
- **Charts**: indigo, banknote green, teal, ochre, slate (`chart-*`), one hue per question; night lifts them (`#8fa4e6`, `#4cc9a4`, `#4fc7bd`, `#e2b24f`, `#7d87a6`).

### Named Rules
**The One Green Note Rule.** Banknote green fills one action per view. Secondary actions are outlined pills on note paper; the seal beside the hero action is printed, not a second button.

**The Foil Is Earned Rule.** The iris foil gradient appears on the gateway prism and its seal and nowhere else: not on buttons, text, backgrounds or borders. Ambient iris light (teal and blue radials at 12-16% alpha) may sit behind a hero; that is light, not foil.

**The Red Means Stop Rule.** Intaglio red marks blocked, failed, destructive and the SPECIMEN overprint. Never decoration, never a highlight.

**The No Violet Rule.** The violet, purple and indigo Tailwind slots resolve to teal iris. Do not add a violet.

## Typography

**Display Font:** Archivo Variable, set at 118% width and weight 640 (with Archivo, ui-sans-serif, system-ui)
**Body Font:** Geist Sans (with ui-sans-serif, system-ui)
**Label/Mono Font:** Geist Mono (with ui-monospace, SFMono-Regular)

**Character:** Archivo stretched wide reads like the lettering of a note's denomination, flat and engraved; Geist is the neutral interface voice; Geist Mono carries the document's machinery: serials, request ids, check labels, code.

### Hierarchy
- **Display** (640, `clamp(2.6rem, 6.2vw, 5.25rem)`, 0.96): the hero headline on the note. The stress clause takes banknote green.
- **Headline** (640, `clamp(2rem, 4vw, 3.25rem)`, 1.02): marketing section headings and the closing seal block. The console overview introduction steps down to `clamp(1.875rem, 2.6vw, 2.625rem)`.
- **Numeral** (640 display, 1.75rem, 1, tabular): figures in the console health ledger; 1.5rem in the marketing console fragment. Provider names in the strip and the wordmark also set in display, at body size.
- **Title** (Geist 600, 0.9375rem, -0.01em): card, panel and register-row titles. All console headings are Geist; the heading font token is the interface face.
- **Body lead** (Geist 400, 1.0625rem, relaxed): the one supporting sentence under a display heading, capped at 48-52ch.
- **Body** (Geist 400, 0.875rem, relaxed): register text, descriptions, table cells.
- **Label** (Geist Mono 400, 0.6875rem, 0.12em, uppercase): check names in a ledger ("KEY", "POLICY"), diagram tags, stamps (SPECIMEN at 0.2em; SAMPLE FIGURES at 0.625rem, 0.16em).
- **Microprint** (Geist Mono 400, 0.5625rem, 0.18em, uppercase): the repeating edge strip of a note. Serial numbers sit at 0.625rem, 0.1em.
- **Code** (Geist Mono, 0.8125rem, relaxed): the terminal panel; inline commands at 0.75rem in a hairline chip.

### Named Rules
**The Denomination Rule.** The wide display face is for marketing h1/h2, the wordmark, provider names and figures read as values. Console titles, card titles and dialog titles are Geist.

**The Microprint Rule.** Mono carries identifiers and machinery (serials, request ids, codes, check labels, stamps). Uppercase tracked mono labels a value inside a ledger, register or diagram; it never sits above a heading as a kicker.

**The Tabular Rule.** Figures that are compared are tabular (tables, stat values, `.tabular`).

## Layout

The marketing page sits in one container (1240px max, 20px gutters, 32px from `sm`). Sections breathe at 96px vertical (128px from `lg`) and are separated by hairline rules or a change of paper (note paper at 60%, or the one night band), never by decorative dividers. Headline columns pair with an artefact in asymmetric two-column grids (`0.75fr / 1.25fr`, `0.8fr / 1.2fr`, `1.12fr / 0.88fr` in the hero) that collapse to one column below `lg`.

The hero is one note spanning the container: microprint strip on top, a serial in the upper-left corner and its twin bottom-right, headline left, the guilloche rosette and prism at right breaking the note's right edge (pushed 22% outward on `lg`), and the printed request ledger as the bottom band (six columns on `lg`, three on `sm`, two on mobile).

The console page pads 16px (24px from `sm`), stacks sections 24px apart, and pairs related panels in 20px-gap two-column grids from `lg`. Overview order is fixed: the health ledger first (one panel, guilloche band on top, six ruled cells), then alerts beside provider health, then the setup checklist or prism band, analytics, top models, recent requests.

Diagrams are drawn to the diagram-design grammar in page tokens as desktop SVGs (min 960px wide, horizontally scrollable with `data-lenis-prevent`) shown from `md`; below `md` the same content is a stacked HTML layout.

### Named Rules
**The Ledger Rule.** Figures read together live in one bordered panel as cells separated by 1px of border colour (a gap-px grid over the border colour), not as separate cards.

**The Register Rule.** Lists of mechanisms, steps or problems are ruled registers (dividers between hairline top and bottom rules, or a table with fixed tiers), not grids of feature cards.

## Elevation & Depth

Flat paper with hairlines by default; shadow is reserved for things that float or are presented as objects. Shadows are indigo-tinted and offset downward with negative spread, so they read as a sheet lifted off the desk rather than a glow. Depth on the prism is real CSS 3D (perspective 900px, plates stacked in `translateZ`).

### Shadow Vocabulary
- **Float** (`box-shadow: 0 1px 2px rgba(20,26,51,0.06), 0 12px 32px -12px rgba(20,26,51,0.2)`; night `0 1px 2px rgba(0,0,0,0.35), 0 16px 40px -16px rgba(0,0,0,0.65)`): menus, selects, popovers, the palette, hovered stage chips.
- **Dialog** (`box-shadow: 0 2px 6px rgba(20,26,51,0.07), 0 32px 64px -24px rgba(20,26,51,0.34)`; night `0 2px 6px rgba(0,0,0,0.4), 0 32px 72px -24px rgba(0,0,0,0.8)`): dialogs, and the presented objects: the hero note, the marketing console fragment, the code terminal.
- **Card hairline** (`box-shadow: 0 1px 2px rgba(20,26,51,0.03)`): the resting card.
- **Pressed ink** on primary buttons (`inset 0 1px 0 rgba(255,255,255,0.12), 0 1px 2px rgba(20,26,51,0.18)`; hero action `inset 0 1px 0 rgba(255,255,255,0.16), 0 8px 18px -8px rgba(13,92,72,0.6)`).

### Named Rules
**The Only What Floats Rule.** Panels, registers and ledgers rest flat on a hairline. Only floating surfaces and the few presented objects (the note, the console fragment, the terminal) carry a shadow.

## Shapes

Softly rounded paper with a sharp stamp. The base radius is 10px, scaled to 5 / 7.5 / 10 / 14 / 18px. Panels and cards round at 18px; the hero note and the closing seal block at 20px; inputs at 10px; stamps at 4px, rotated -3deg when they are an overprint (SPECIMEN). Every action and badge is a full pill. The brand geometry is circular: the seal is a ring around a dashed inner guilloche ring with a T drawn as a toll barrier, and guilloche rosettes are computed closed curves (an 18-wave outer ring in seven phase copies, a 12-wave inner ring in six), stroked at 0.4-0.6 in the guilloche line colours. Prism plates round at 22%.

## Components

### Buttons
Tactile pills that press in.
- **Shape:** full pill (9999px). Heights 28 / 32 / 36 / 44px (`xs` / `sm` / default / `lg`); marketing actions 48px.
- **Primary:** banknote green with paper-on-green text, a pressed-ink inner highlight, 16px horizontal padding (28-32px on marketing). Hover deepens to `banknote-green-deep`.
- **Hover / Focus:** 150ms on `cubic-bezier(0.23, 1, 0.32, 1)` for colour, border and shadow; press scales to 0.97; focus is a 3px teal-iris ring at 35-40%. Trailing arrows nudge 2px on hover.
- **Outline:** note paper, strong hairline, ink text; hover fills paper press. The secondary action everywhere.
- **Ghost:** ink at 80%, paper-press fill on hover. **Destructive:** intaglio red text on a 10% red fill (20% on night). **Link:** green text, underline on hover at 4px offset.

### Chips and Stamps
- **Badge:** 24px pill, paper-shade fill with hairline (secondary) or primary fill (default); destructive is red on 10% red.
- **SPECIMEN stamp:** mono 0.6875rem at 0.2em, intaglio red text on a half-strength red border, 4px radius, rotated -3deg. Placed on illustrative request ledgers.
- **SAMPLE FIGURES stamp:** mono 0.625rem at 0.16em, muted ink on a strong hairline, 4px radius, upright. Placed in the header of illustrative console fragments.

### Cards / Containers
- **Corner Style:** 18px panels; 20px notes.
- **Background:** note paper on security paper; paper shade for header bands.
- **Shadow Strategy:** flat (see Elevation); dialog shadow only for presented objects.
- **Border:** hairline; strong hairline for the ledger panel, the hero note and the problem register.
- **Internal Padding:** 20px (16px for the small size); card footers sit on paper shade at 45% behind a hairline.

### Inputs / Fields
- **Style:** 36px tall, 10px radius, input-strength hairline, note-paper fill (input colour at 20% on night), teal-iris caret.
- **Focus:** border turns teal iris with a 3px ring at 25%. Hover strengthens the border.
- **Error / Disabled:** red border with a 3px red ring at 20%; disabled at 50% opacity on an input-tinted fill.

### Navigation
- **Console sidebar:** security-paper ground, muted ink items, `paper-press-active` fill and full ink for the current page (`aria-current="page"`), a teal-iris dot on the collapsed rail. Items press to 0.95.
- **Topbar:** the command palette trigger is a 36px note-paper pill with a hairline; the palette opens with zero-duration transitions.
- **Marketing:** icon controls are 36px round ghost buttons; the wordmark is the seal plus "Tollseal" in display at 1.0625rem.

### Health Ledger (signature)
One panel with a strong hairline, a guilloche band across its top, a Geist title row ("Health", with a green telemetry dot), then six cells (two columns, three from `md`, six from `xl`) in a gap-px grid. Each cell is a link: teal-iris icon and label, the figure in display numerals, a supporting line with a coloured delta (green when better, red when worse; spend, latency and errors are worse when they rise). A figure with nothing behind it is a ghost numeral at 25% ink with "Nothing in this range"; a missing measurement is an em dash. Cells fill paper press at 60% on hover.

### The Note (signature)
The hero document: note paper, strong hairline, 20px radius, dialog shadow, unclipped so the rosette breaks its right edge. Microprint strip on top, serials in opposite corners, and a bottom band on security paper at 60% carrying a guilloche band, the request id, the SPECIMEN stamp and the six-check ledger (mono uppercase check name, Geist 600 result with a status icon, mono detail).

### Gateway Prism (signature)
Four foil plates stacked in CSS 3D (`rotateX(56deg) rotateZ(-44deg)`, lower plates at 32 / 50 / 70% opacity) under an intaglio cap with a gold edge that carries the seal in gold, over a soft teal glow, inside a guilloche rosette. It floats 8px on a 7s ease-in-out loop that stops under reduced motion. Decorative and `aria-hidden`.

### Code Terminal
Always dark: terminal ink ground, a blue hairline at 18%, 18px radius, dialog shadow. CLI / SDK / REST tabs as 10%-white pills, a 32px round copy button with a polite live region, teal-iris focus rings.

### Motion
- **Tokens:** `--ease-out` / `--ease-out-strong` `cubic-bezier(0.23, 1, 0.32, 1)`, `--ease-drawer` `cubic-bezier(0.32, 0.72, 0, 1)`, `--ease-micro` `cubic-bezier(0.5, 0, 0.5, 1)`; durations 140 / 220 / 480ms.
- **Hero:** under `gsap.matchMedia("(prefers-reduced-motion: no-preference)")` only, the hero guilloche engraves once (stroke-dashoffset, 1.6s `power2.inOut`, 0.03s stagger) and the six ledger checks stamp in from 0.5s (0.5s `expo.out`, 0.16s stagger, from 6px and 0.97 scale). Then everything rests.
- **Diagrams:** the diagram-design animation contract and its canonical controller. The complete diagram is in the markup; only `.motion-ready` may dim items (12% opacity, 8px). Reveal once on first view; Play / Pause / Replay / Prev / Next controls; step 480ms, hold 720ms, ease `cubic-bezier(.2, .8, .2, 1)`. Reduced motion and print show the static complete frame with controls and decorative items hidden.
- **Flow token:** one dot travels the headline path of "Where Tollseal sits" on a 4s loop, hidden behind the gateway box, running only on hover or focus-within.
- **Never:** no scroll-reveal of sections; surfaces opened from the keyboard (the command palette) do not animate.

## Do's and Don'ts

### Do:
- **Do** keep banknote green to one filled action per view, and print the seal in the primary.
- **Do** draw guilloche only from the computed `GuillocheRosette` and `GuillocheBand`, in the guilloche line tokens, `aria-hidden`.
- **Do** stamp every illustrative figure: SPECIMEN on sample requests, SAMPLE FIGURES on sample console fragments.
- **Do** set compared figures in display numerals with tabular figures, inside one gap-px ledger panel.
- **Do** show empty figures as a ghost numeral with "Nothing in this range", and missing measurements as an em dash.
- **Do** use ruled registers (dividers between hairlines) for lists of mechanisms, problems and steps.
- **Do** ship every surface complete without JavaScript; motion only engraves, stamps or steps through what is already rendered, under `gsap.matchMedia` or the diagram controller.
- **Do** provide the stacked HTML version of every diagram below `md`.

### Don't:
- **Don't** put the iris foil on anything but the gateway prism and its seal.
- **Don't** use intaglio red for anything but blocked, failed, destructive and the SPECIMEN overprint.
- **Don't** invent customers, traffic totals, logos or certifications, or show unlabelled sample figures.
- **Don't** add a violet; the violet slots are teal iris.
- **Don't** set console, card or dialog titles in the display face.
- **Don't** build feature-card grids or a dark glowing dev-tool hero; the one night band on the marketing page is the developers band.
- **Don't** reveal sections on scroll, or animate surfaces opened from the keyboard.
- **Don't** shadow resting panels; shadow is for floating surfaces and presented objects.
