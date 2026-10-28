"use client";

import { ScrollReveal } from "@/components/landing/scroll-reveal";
import { Container, SectionLabel } from "@/components/landing/section";
import { PILLARS } from "@/lib/landing-data";

export function Pillars() {
  return (
    <section
      className="relative overflow-hidden border-b border-[rgba(111,78,55,0.10)]"
      style={{ background: "var(--canvas)" }}
    >
      <Container className="py-24 sm:py-32 lg:py-40">
        {/* Section label + heading — left-aligned */}
        <ScrollReveal>
          <div className="max-w-xl">
            <SectionLabel>{PILLARS.eyebrow}</SectionLabel>
            <h2
              className="mt-4 font-display text-[clamp(2.5rem,5vw,4rem)] font-semibold text-[#1C1412] leading-[0.95]"
              style={{ fontFamily: "var(--font-cormorant)", letterSpacing: "-0.04em" }}
            >
              {PILLARS.headline}
            </h2>
            <p className="mt-6 text-base leading-relaxed text-[#4A3A30] max-w-[48ch]" style={{ letterSpacing: "-0.01em" }}>
              {PILLARS.body}
            </p>
          </div>
        </ScrollReveal>

        {/* 2×2 grid: each pillar in an editorial cell — no card surface */}
        <ul className="mt-20 grid gap-0 md:grid-cols-2 border-t border-l border-[rgba(111,78,55,0.10)]">
          {PILLARS.items.map((pillar, index) => (
            <li
              key={pillar.title}
              className="border-b border-r border-[rgba(111,78,55,0.10)] p-8 sm:p-10"
            >
              <ScrollReveal delay={index * 0.09}>
                {/* Large ghost number — editorial accent */}
                <div
                  className="font-display text-[5rem] font-semibold leading-none text-[rgba(111,78,55,0.08)] tabular-nums select-none mb-4"
                  style={{ fontFamily: "var(--font-cormorant)", letterSpacing: "-0.05em" }}
                  aria-hidden
                >
                  {pillar.index}
                </div>

                <h3
                  className="text-2xl font-semibold text-[#1C1412]"
                  style={{ fontFamily: "var(--font-cormorant)", letterSpacing: "-0.03em" }}
                >
                  {pillar.title}
                </h3>
                <p className="mt-3 text-base leading-relaxed text-[#4A3A30] max-w-[40ch]">
                  {pillar.body}
                </p>

                {/* Points: clean horizontal chips */}
                <ul className="mt-6 flex flex-wrap gap-2">
                  {pillar.points.map((point) => (
                    <li
                      key={point}
                      className="border border-[rgba(111,78,55,0.20)] px-3 py-1 text-xs font-medium text-[#4A3A30]"
                      style={{ fontFamily: "var(--font-fira-code)" }}
                    >
                      {point}
                    </li>
                  ))}
                </ul>
              </ScrollReveal>
            </li>
          ))}
        </ul>
      </Container>
    </section>
  );
}
