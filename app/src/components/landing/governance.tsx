"use client";

import { ScrollReveal } from "@/components/landing/scroll-reveal";
import { Container, SectionLabel } from "@/components/landing/section";
import { GOVERNANCE } from "@/lib/landing-data";

/**
 * Dark espresso section — rich inverted palette.
 * Governance capabilities as editorial list + evidence metrics.
 */
export function Governance() {
  return (
    <section
      className="relative overflow-hidden border-b border-[rgba(229,228,226,0.08)] grain"
      style={{ background: "#1C1412" }}
    >
      <Container className="py-24 sm:py-32 lg:py-40">
        {/* Z-pattern: text left (4), capabilities right (8) */}
        <div className="grid gap-16 lg:grid-cols-12 lg:gap-20">

          {/* Left: heading */}
          <div className="lg:col-span-4">
            <ScrollReveal>
              <SectionLabel className="text-[rgba(230,224,240,0.55)]">{GOVERNANCE.eyebrow}</SectionLabel>
              <h2
                className="mt-4 font-display text-[clamp(2.5rem,5vw,4rem)] font-semibold leading-[0.95]"
                style={{
                  fontFamily: "var(--font-cormorant)",
                  letterSpacing: "-0.04em",
                  color: "#E5E4E2",
                }}
              >
                {GOVERNANCE.headline}
              </h2>
              <p
                className="mt-6 text-base leading-relaxed max-w-[40ch]"
                style={{ color: "rgba(229,228,226,0.60)", letterSpacing: "-0.01em" }}
              >
                {GOVERNANCE.body}
              </p>

              {/* Accent rule */}
              <div
                className="mt-10 h-px"
                style={{
                  background: "linear-gradient(90deg, rgba(230,224,240,0.30), transparent)",
                  maxWidth: "160px",
                }}
                aria-hidden
              />
            </ScrollReveal>
          </div>

          {/* Right: capability list */}
          <ul className="lg:col-span-8 space-y-0">
            {GOVERNANCE.capabilities.map((item, index) => (
              <li
                key={item.title}
                className="border-t border-[rgba(229,228,226,0.08)]"
              >
                <ScrollReveal delay={index * 0.09}>
                  <div className="py-8 grid gap-6 sm:grid-cols-[3rem_1fr] sm:gap-8 items-start">
                    <div
                      className="font-mono text-xs font-medium pt-1"
                      style={{
                        fontFamily: "var(--font-fira-code)",
                        letterSpacing: "0.08em",
                        color: "rgba(229,228,226,0.35)",
                      }}
                    >
                      {String(index + 1).padStart(2, "0")}
                    </div>
                    <div>
                      <h3
                        className="text-xl font-semibold"
                        style={{
                          fontFamily: "var(--font-dm-sans)",
                          letterSpacing: "-0.02em",
                          color: "#E5E4E2",
                        }}
                      >
                        {item.title}
                      </h3>
                      <p
                        className="mt-3 text-sm leading-relaxed max-w-[52ch]"
                        style={{ color: "rgba(229,228,226,0.55)" }}
                      >
                        {item.body}
                      </p>
                    </div>
                  </div>
                </ScrollReveal>
              </li>
            ))}
            <li className="border-t border-[rgba(229,228,226,0.08)]" />
          </ul>
        </div>

        {/* Evidence metrics — 4-column stat grid */}
        <ScrollReveal delay={0.25} className="mt-20 pt-12 border-t border-[rgba(229,228,226,0.08)]">
          <dl className="grid gap-8 sm:grid-cols-2 lg:grid-cols-4 sm:gap-0">
            {GOVERNANCE.evidence.map((item, i) => (
              <div
                key={item.label}
                className="sm:border-r sm:border-[rgba(229,228,226,0.08)] last:border-r-0 sm:pr-8 sm:pl-8 first:pl-0"
              >
                <dt
                  className="text-[10px] uppercase"
                  style={{
                    fontFamily: "var(--font-fira-code)",
                    letterSpacing: "0.12em",
                    color: "rgba(229,228,226,0.40)",
                  }}
                >
                  {item.label}
                </dt>
                <dd
                  className="mt-2 text-base font-semibold"
                  style={{
                    fontFamily: "var(--font-dm-sans)",
                    letterSpacing: "-0.02em",
                    color: "#E5E4E2",
                  }}
                >
                  {item.value}
                </dd>
              </div>
            ))}
          </dl>
        </ScrollReveal>
      </Container>
    </section>
  );
}
