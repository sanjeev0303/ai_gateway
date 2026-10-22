"use client";

import { useRef, useState } from "react";
import { ScrollReveal } from "@/components/landing/scroll-reveal";
import { Container, SectionLabel } from "@/components/landing/section";
import { USE_CASES } from "@/lib/landing-data";
import { cn } from "@/lib/utils";

export function UseCases() {
  const cases = USE_CASES.cases;
  const [activeId, setActiveId] = useState(cases[0]?.id ?? "");
  const tabRefs = useRef<Record<string, HTMLButtonElement | null>>({});
  const active = cases.find((item) => item.id === activeId) ?? cases[0];

  function onKeyDown(event: React.KeyboardEvent<HTMLButtonElement>, index: number) {
    const offsets: Record<string, number> = { ArrowRight: 1, ArrowLeft: -1 };
    const delta = offsets[event.key];
    if (delta === undefined) return;
    event.preventDefault();
    const next = cases[(index + delta + cases.length) % cases.length];
    if (!next) return;
    setActiveId(next.id);
    tabRefs.current[next.id]?.focus();
  }

  if (!active) return null;

  return (
    <section
      id="solutions"
      className="relative overflow-hidden border-b border-[rgba(111,78,55,0.10)] grain scroll-mt-14"
      style={{ background: "var(--canvas)" }}
    >
      <Container className="py-24 sm:py-32 lg:py-40">
        <ScrollReveal>
          <SectionLabel>{USE_CASES.eyebrow}</SectionLabel>
          <h2
            className="mt-4 font-display text-[clamp(2.5rem,5vw,4rem)] font-semibold text-[#1C1412] leading-[0.95]"
            style={{ fontFamily: "var(--font-cormorant)", letterSpacing: "-0.04em" }}
          >
            {USE_CASES.headline}
          </h2>
          <p className="mt-6 text-base leading-relaxed text-[#4A3A30] max-w-[48ch]" style={{ letterSpacing: "-0.01em" }}>
            {USE_CASES.body}
          </p>
        </ScrollReveal>

        <ScrollReveal delay={0.15} className="mt-14">
          {/* Tab list — inline underline style */}
          <div
            role="tablist"
            aria-label="Persona"
            className="flex border-b border-[rgba(111,78,55,0.12)]"
          >
            {cases.map((item, index) => {
              const selected = item.id === active.id;
              return (
                <button
                  key={item.id}
                  type="button"
                  role="tab"
                  id={`usecase-tab-${item.id}`}
                  aria-selected={selected}
                  aria-controls="usecase-panel"
                  tabIndex={selected ? 0 : -1}
                  ref={(node) => { tabRefs.current[item.id] = node; }}
                  onClick={() => setActiveId(item.id)}
                  onKeyDown={(e) => onKeyDown(e, index)}
                  className={cn(
                    "relative px-6 py-3 text-sm font-medium transition-colors duration-150 focus-visible:outline-2 focus-visible:outline-[#6F4E37]",
                    selected ? "text-[#6F4E37]" : "text-[#7A6458]"
                  )}
                  style={{ letterSpacing: "-0.01em" }}
                >
                  {item.label}
                  {selected && (
                    <span
                      className="absolute inset-x-0 -bottom-px h-0.5"
                      style={{ background: "#6F4E37" }}
                      aria-hidden
                    />
                  )}
                </button>
              );
            })}
          </div>

          {/* Panel — Z-pattern: text left, data surface right */}
          <div
            id="usecase-panel"
            role="tabpanel"
            aria-labelledby={`usecase-tab-${active.id}`}
            tabIndex={0}
            className="mt-10 grid gap-12 lg:grid-cols-[1.4fr_1fr] lg:gap-20 focus-visible:outline-none"
          >
            <div>
              <h3
                className="text-3xl font-semibold text-[#1C1412] leading-tight"
                style={{ fontFamily: "var(--font-cormorant)", letterSpacing: "-0.03em" }}
              >
                {active.headline}
              </h3>
              <p className="mt-4 text-base leading-relaxed text-[#4A3A30] max-w-[44ch]" style={{ letterSpacing: "-0.01em" }}>
                {active.body}
              </p>
              <ul className="mt-8 space-y-3">
                {active.outcomes.map((outcome) => (
                  <li key={outcome} className="flex items-start gap-3 text-sm text-[#4A3A30]">
                    <div
                      className="mt-2 h-px w-4 shrink-0"
                      style={{ background: "#6F4E37" }}
                      aria-hidden
                    />
                    {outcome}
                  </li>
                ))}
              </ul>
            </div>

            {/* Right: surface data as a definition list */}
            <div
              className="border border-[rgba(111,78,55,0.16)] p-6 self-start"
              style={{ background: "rgba(230,224,240,0.25)" }}
            >
              <dl className="space-y-0">
                {active.surface.map((row) => (
                  <div
                    key={row.label}
                    className="flex items-baseline justify-between py-4 border-b border-[rgba(111,78,55,0.08)] last:border-b-0"
                  >
                    <dt className="text-xs text-[#7A6458]" style={{ fontFamily: "var(--font-fira-code)" }}>
                      {row.label}
                    </dt>
                    <dd
                      className="text-sm font-semibold text-[#1C1412] text-right"
                      style={{ letterSpacing: "-0.02em" }}
                    >
                      {row.value}
                    </dd>
                  </div>
                ))}
              </dl>
            </div>
          </div>
        </ScrollReveal>
      </Container>
    </section>
  );
}
