# Design System: AI Governance (apps/web marketing site)

Committed design direction for the public marketing site (landing) and the
in-app documentation routes. The dashboard (`(dashboard)`) and auth
(`(auth)`) surfaces keep their existing minimal styles and are out of scope
for this document; they may be folded in during a later milestone.

## 0. Research Log

Greenfield: no prior marketing design exists (stub page only).

- **Embedded refs shortlisted**: `vercel.md`, `soft-skill.md` (frontend skill
  design library). Stripe/Linear (color-first, accent-forward) were rejected
  because the user explicitly banned purple accents; Vercel's monochrome
  precision fits a developer-infrastructure product and the ban list exactly.
- **Lazyweb**: skipped. No external site research needed for a
  developer-infrastructure landing; the committed direction below is grounded
  in the Vercel reference system and the product's own technical identity
  (Go proxy, policy snapshots, audit-grade usage).
- **Imagen drafts**: skipped lane. No user-supplied reference image exists,
  and the hero focal object (request hot-path terminal card) is rendered in
  HTML/CSS, which stays crisp at every viewport and avoids image weight.
- **react-scan / react-doctor**: **accepted debt**, see §8. This marketing
  route is static server-rendered markup with zero client interactivity
  beyond `<a>` navigation; render-profiling tooling adds noise without
  signal here and would pollute the monorepo devDependencies. Revisit with
  the dashboard milestone, where client state appears.

## 1. Atmosphere & Identity

**Direction (committed)**: *A quiet governance console.* Light, near-white
canvas; ink-black text; one restrained blue used only for links and focus
affordances. The signature object is a dark terminal card tracing a request
through the gateway hot path: `authn → policy snapshot → rate/budget →
forward → stream → usage`. Precision over decoration.

This is a developer-infrastructure product (a Go proxy with compiled policy
snapshots and audit-grade usage records). The design must read as
engineering-trustworthy, not consumer-playful. Typography and a hard, fine
border system carry the personality; color stays out of the way.

**Do**: monochrome dominance, Geist Sans with tight tracking, mono labels for
technical annotation, shadow-as-border surfaces, restrained motion.
**Don't**: purple gradients, pill-shaped buttons, emoji icons, em dashes in
copy, fake metrics/reviews/customers, vague hero copy, cursor effects,
scroll-driven spectacle, AI-slop phrasing.

## 2. Color

Monochrome system with a single reserved accent. No hue beyond blue.

| Token | Value | Use |
|---|---|---|
| `--canvas` | `#ffffff` | page background |
| `--canvas-subtle` | `#fafafa` | section alternation, terminal card well |
| `--ink` | `#171717` | primary text, primary button |
| `--ink-secondary` | `#4d4d4d` | secondary text (passes AA on white, 8.2:1) |
| `--ink-tertiary` | `#666666` | captions, meta (passes AA, 5.2:1) |
| `--line` | `rgba(0,0,0,0.08)` | shadow-as-border on light surfaces |
| `--line-strong` | `rgba(0,0,0,0.14)` | keyboard-focus, active states |
| `--surface` | `#ffffff` | cards, panels |
| `--surface-dim` | `#fafafa` | nested wells inside cards |
| `--link` | `#0072f5` | inline links, focus ring (Link Blue) |
| `--link-subtle` | `#1f1f1f` | nav links (ink, darkens to black on hover) |
| `--terminal-bg` | `#0d0d0d` | hero terminal card |
| `--terminal-text` | `#d4d4d4` | terminal foreground |
| `--terminal-dim` | `#6e6e6e` | terminal muted lines |
| `--terminal-ok` | `#3fb950` | status check glyphs (GitHub scale, terminal context only) |
| `--terminal-warn` | `#d29922` | pending/limit glyphs (terminal context only) |

The gradient ban is absolute: no `linear-gradient`/`radial-gradient` on the
marketing site, including text, buttons, and cards. Status hue appears only
inside the terminal mock (a synthesized log output), never as UI chrome.

## 3. Typography

Geist Sans (body, UI, headings) and Geist Mono (technical annotation) from the
`geist` npm package via `next/font` (variable weights, self-hosted, zero
layout shift, zero external font requests).

| Role | Style |
|---|---|
| Display | Geist Sans 600, `48px`/`1.05`, tracking `-0.05em` |
| H1 (section) | Geist Sans 600, `40px`/`1.1`, tracking `-0.04em` |
| H2 | Geist Sans 600, `32px`/`1.15`, tracking `-0.03em` |
| Lead | Geist Sans 400, `20px`/`1.5` |
| Body | Geist Sans 400, `16px`/`1.6` |
| Caption | Geist Sans 500, `14px`/`1.4` |
| Mono label | Geist Mono 400, `13px`, uppercase, tracking `0.04em` |
| Mono body | Geist Mono 400, `13px`/`1.5` (terminal card) |
| Code/literal | Geist Mono 400, `13px`, inline `--surface-dim` well |

Weights used: 400, 500, 600. No italics. No em dash (`—`) anywhere in copy
or UI strings.

## 4. Spacing & Layout

- Base unit `8px`; scale: `4 8 12 16 24 32 48 64 96 128`.
- Content column `max-width: 1200px` (`--content-max`), `padding-inline: 24px`,
  centered; text-heavy columns cap at `42rem-56rem` for measure.
- Section rhythm: `padding-block: 96px` on desktop, `72px` or `64px` on
  mobile. Vertical whitespace is the primary separator; hairlines (`1px`
  shadow) handle secondary separation.
- Grid: 12-col for marketing grids collapsing to 1-col at `≤896px`. Card
  min-width guards against cramping.
- Nav: sticky, `--canvas` backdrop, hairline bottom border; height `64px`.

## 5. Components

All primitives use **shadow-as-border** (`box-shadow: 0px 0px 0px 1px
rgba(0,0,0,0.08)`), never `border`, so interactive hover/active states can
shift the shadow stack without layout jitter. Radius: `8px` cards, `6px`
buttons/inputs, `4px` tags.

**Button**
- `primary`: ink `#171717` bg, white text, radius `6px`, `height 40px`,
  `padding 0 20px`, Geist 500 `14px`, hover `#000`; translateY(-1px) with
  shadow on hover. Active: translateY(0).
- `secondary`: `--surface` bg, shadow-as-border; hover: hairline darkens to
  `0.14`.
- `link`: ink text; hover underline offset `2px`.
- All: `:focus-visible` 2px solid `--link` ring offset 2px. No pill radii.

**Cards** (`--surface`): shadow-as-border; inner vertical padding `24px`;
optional `--surface-dim` nested wells for mock content. Hover: shadow stack
lift (`0 0 0 1px` → `0 1px 2px` + `0 2px 8px`) with 150ms ease-out. Cards
are informational; only interactive cards raise.

**Terminal card** (hero focal object): `--terminal-bg`, mono 13px lines,
three window dots in muted gray, `OK`/`LIMIT`/`AUDIT` glyph columns in
`--terminal-ok`/`--terminal-warn`. Represents the hot-path sequence. Lines are
static markup, never typed, and no caret or cursor animation. The one authored
motion in the product is a single one-shot reveal of this card on load: each
line and gate enters once via `transform`/`opacity` only, staggered by a
`--d` delay in documented enforcement order, then stops permanently. No
scroll-linked, parallax, or cursor-tracking motion anywhere.

**Eyebrow**: mono 13px uppercase `--ink-secondary`, letter-spacing `0.04em`,
placed above display headings ("DATA PLANE", "POLICY SNAPSHOTS").

**Nav**: brand wordmark (Geist 600, ink) + anchor links (Product, Pricing,
Docs) + right cluster (Sign in link, Get started primary button). Mobile:
anchor list collapses into the sign-in/get-started cluster `<details>`
disclosure, no JS.

**Pricing section**: an enforcement control matrix plus two deployment plans.
The matrix is the primary object: one row per control, each with a plain-language
summary, the exact accepted values as mono chips, and the defining type or file
as a source reference (`domain.RuleType*`, `DefaultGatewayConfig`). Three
columns on desktop (what / values / source), collapsing to one stacked column
below 896px. Below it, two plans: Hosted marked "Not yet available" in
`--ink-tertiary`, and Self-hosted as the primary card. Feature truthfulness:
every value ties to shipped repository capability (`backend`,
`migrations/00007_usage.sql`, `docs/adr`). Unbuilt capability is stated as
unbuilt in the row itself, never as a number. No invented tiers, prices, seat
counts, or retention windows; the `plans`, `subscription`, and `billing_events`
tables carry no seeded rows, so there is nothing to quote.

**Footer**: hairline top; brand + short mission line; columns: Product,
Docs, Company(legal links to repo). `© YYYY` line.

## 6. Motion & Interaction

Restrained; emphasis on state clarity over spectacle (scroll animations
banned).

| Token | Value |
|---|---|
| `--ease-out` | `cubic-bezier(0.16, 1, 0.3, 1)` |
| `--ease-micro` | `cubic-bezier(0.5, 0, 0.5, 1)` |
| micro (`hover/active/subtle`) | 120ms, `--ease-micro` |
| standard | 250ms, `--ease-out` |
| emphasis (hero load-in) | 400ms, `--ease-out` |

Rules:

- Animate **only** `transform` and `opacity` (GPU-composited, no layout/paint
  thrash).
- One hero load-in on entry: terminal card `opacity 0 → 1` with `translateY(8px
  → 0)`, `--ease-out` 400ms; nothing else animates on scroll.
- Hover/active translate and shadow lifts ≤ 2px.
- `@media (prefers-reduced-motion: reduce)`: all animation and transitions
  collapse to 0 duration; load-in renders fully visible.

## 7. Depth & Surface

- Flat, light-first system. Depth is **borders, not shadows** for resting
  state (shadow-as-border), so the page stays crisp and print/overlay-safe.
- Elevation = stacked shadows, used only on interactive hover/focus and the
  sticky nav:
  - `--shadow-1`: `0 0 0 1px rgba(0,0,0,0.08), 0 1px 2px rgba(0,0,0,0.04)`
  - `--shadow-2`: `0 0 0 1px rgba(0,0,0,0.08), 0 2px 4px rgba(0,0,0,0.04),
    0 4px 12px rgba(0,0,0,0.06)`
- Terminal card is the single dark surface; it sits directly on `--canvas`
  with `--shadow-2` to anchor the hero.

## 8. Accessibility & Accepted Debt

- Contrast: all text pairs verified ≥ 4.5:1 AA (primary/secondary/tertiary
  inks on white and `--surface-dim`; `--link` 4.5:1 on white). Terminal uses
  `#d4d4d4` on `#0d0d0d` (12.9:1).
- Focus: `:focus-visible` visible 2px `--link` ring, offset 2px, on all
  interactive elements. Nav disclosure keyboard-operable.
- Motion: `prefers-reduced-motion` support mandatory (§6).
- Semantics: single `<main>`, landmark `<header>/<nav>/<footer>`, heading
  hierarchy `<h1>` per page, real anchors (`<a>`) for all navigation, no
  client JS required anywhere on the marketing route.
- Copy: no em dashes, no fake metrics/counters/reviews, no superlatives
  without a repository-backed basis (architecture, ADRs, milestones are the
  source of truth).
- **Accepted debt**: react-scan/react-doctor not installed (static RSC page,
  zero client JS; revisit at dashboard milestone). Imagegen/UI-research lanes
  skipped (§0). No automated a11y snapshot tests yet; the perfection gate
  (Lighthouse 100 mobile + desktop) is the enforced quality bar for this
  route.