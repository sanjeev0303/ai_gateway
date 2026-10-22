# Task: Sentinel Homepage Complete Redesign

## Status: COMPLETE ✓

## What was built
Full homepage redesign — 17 sections + footer + nav.
Theme: #6F4E37 (coffee 30%), #E5E4E2 (canvas 60%), #E6E0F0 (lavender 10%).

## Design system
- Cormorant Garamond: display/editorial headlines (tight -0.04em tracking)
- DM Sans: body copy
- Fira Code: labels, mono values, code
- 60-30-10 color distribution across all sections
- Z-pattern layouts (alternating text-left/visual-right)
- Rule of thirds accent lines
- Natural linen grain + weave textures (CSS SVG filter)
- Lenis smooth scroll + GSAP ScrollTrigger + Motion scroll reveals
- Zero rounded corners, no drop shadows, no Lucide icons in sections

## Sections completed
- [x] globals.css — typography tokens, grain/weave, reveal utilities, stat-block
- [x] motion.ts — GSAP easing strings + Motion/React variants
- [x] use-gsap-reveal.ts hook
- [x] lenis-provider.tsx — Lenis + GSAP ScrollTrigger integrated
- [x] layout.tsx — LenisProvider + ThemeProvider wired
- [x] dashboard-preview.tsx — brown palette, sharp borders
- [x] hero.tsx — Z-grid, overlap ghost text, stat callouts, Cormorant headline
- [x] provider-strip.tsx — marquee, edge fade masks
- [x] problem.tsx — sticky left heading, scrolling numbered list
- [x] gateway-flow.tsx — lavender bg, pipeline nodes, loop steps
- [x] pillars.tsx — 2×2 grid, ghost index numbers, tag chips
- [x] request-inspector.tsx — tabbed panel, left-border trace, Z-reverse
- [x] use-cases.tsx — underline tabs, definition list surface
- [x] attribution.tsx — horizontal bar chart, metric toggle
- [x] governance.tsx — dark espresso section, numbered capability list
- [x] code-tabs.tsx — dark code panel, no terminal chrome
- [x] architecture.tsx — lavender bg, vertical timeline, payload table
- [x] comparison.tsx — editorial table, dot accent cells
- [x] trust.tsx — 2×2 grid, accent line per item
- [x] pricing-teaser.tsx — numbered list, deployment options, CTA panel
- [x] final-cta.tsx — dark, giant headline, Z-split
- [x] footer.tsx — 5-column, dark espresso, status badge
- [x] nav.tsx — transparent→frosted scroll behavior, mobile drawer

## Verification
- HTTP 200 on / and /pricing
- npx tsc --noEmit: 0 errors
- No onMouseEnter in server components
