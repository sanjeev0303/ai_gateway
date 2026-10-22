"use client";

import { ScrollReveal } from "@/components/landing/scroll-reveal";
import { Container, SectionLabel } from "@/components/landing/section";
import { PROBLEM } from "@/lib/landing-data";

export function Problem() {
  return (
    <section
      id="product"
      className="relative overflow-hidden border-b border-[rgba(111,78,55,0.10)] weave grain scroll-mt-14"
      style={{ background: "var(--canvas)" }}
    >
      <Container className="py-24 sm:py-32 lg:py-40">
        {/* Z-layout: text left (5 cols), visual right (7 cols) */}
        <div className="grid gap-16 lg:grid-cols-12 lg:gap-20 lg:items-start">

          {/* Left: sticky heading */}
          <div className="lg:col-span-5 lg:sticky lg:top-32">
            <ScrollReveal>
              <SectionLabel>{PROBLEM.eyebrow}</SectionLabel>
              <h2
                className="mt-4 font-display text-[clamp(2.5rem,5vw,4rem)] font-semibold text-[#1C1412] leading-[0.95]"
                style={{ fontFamily: "var(--font-cormorant)", letterSpacing: "-0.04em" }}
              >
                {PROBLEM.headline}
              </h2>
              <p className="mt-6 text-base leading-relaxed text-[#4A3A30] max-w-[42ch]" style={{ letterSpacing: "-0.01em" }}>
                {PROBLEM.body}
              </p>
            </ScrollReveal>

            {/* Rule of thirds accent */}
            <div
              className="mt-12 h-px"
              style={{ background: "linear-gradient(90deg, #6F4E37 0%, rgba(111,78,55,0.10) 60%, transparent 100%)", maxWidth: "180px" }}
              aria-hidden
            />
          </div>

          {/* Right: stacked problem cards — editorial list style */}
          <ul className="lg:col-span-7 space-y-0">
            {PROBLEM.points.map((point, index) => {
              return (
                <li
                  key={point.title}
                  className="border-t border-[rgba(111,78,55,0.10)]"
                >
                  <ScrollReveal delay={index * 0.1}>
                    <div className="py-8 grid gap-6 sm:grid-cols-[3rem_1fr] sm:gap-8 items-start">
                      {/* Index number */}
                      <div
                        className="font-mono text-xs font-medium text-[#7A6458] pt-1"
                        style={{ fontFamily: "var(--font-fira-code)", letterSpacing: "0.08em" }}
                      >
                        {String(index + 1).padStart(2, "0")}
                      </div>

                      <div>
                        <h3
                          className="text-xl font-semibold text-[#1C1412]"
                          style={{ fontFamily: "var(--font-dm-sans)", letterSpacing: "-0.02em" }}
                        >
                          {point.title}
                        </h3>
                        <p className="mt-3 text-sm leading-relaxed text-[#4A3A30] max-w-[52ch]">
                          {point.body}
                        </p>
                      </div>
                    </div>
                  </ScrollReveal>
                </li>
              );
            })}
            <li className="border-t border-[rgba(111,78,55,0.10)]" />
          </ul>
        </div>
      </Container>
    </section>
  );
}
