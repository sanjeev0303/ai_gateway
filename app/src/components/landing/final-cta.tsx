"use client";

import Link from "next/link";
import { ScrollReveal } from "@/components/landing/scroll-reveal";
import { Container } from "@/components/landing/section";
import { FINAL_CTA } from "@/lib/landing-data";

export function FinalCta() {
  return (
    <section
      className="relative overflow-hidden border-b border-[rgba(229,228,226,0.08)] grain"
      style={{ background: "#1C1412" }}
    >
      {/* Subtle lavender ambient — Z-top-right corner */}
      <div
        className="pointer-events-none absolute top-0 right-0 w-1/2 h-full"
        style={{
          background:
            "radial-gradient(ellipse 80% 60% at 100% 0%, rgba(230,224,240,0.08) 0%, transparent 70%)",
        }}
        aria-hidden
      />

      <Container className="relative z-10 py-28 sm:py-36 lg:py-44">
        <div className="grid gap-12 lg:grid-cols-12 lg:items-end">

          {/* Left: large editorial headline */}
          <div className="lg:col-span-7">
            <ScrollReveal>
              <p
                className="font-mono text-xs uppercase text-[rgba(230,224,240,0.65)] font-semibold mb-6"
                style={{ fontFamily: "var(--font-fira-code)", letterSpacing: "0.14em" }}
              >
                Ready to get started
              </p>
              <h2
                className="font-display text-[clamp(3rem,7vw,6rem)] font-semibold leading-[0.92]"
                style={{
                  fontFamily: "var(--font-cormorant)",
                  letterSpacing: "-0.04em",
                  color: "#E5E4E2",
                }}
              >
                {FINAL_CTA.headline}
              </h2>
            </ScrollReveal>
          </div>

          {/* Right: body + CTAs */}
          <div className="lg:col-span-5">
            <ScrollReveal delay={0.15}>
              <p
                className="text-base leading-relaxed max-w-[42ch]"
                style={{ color: "rgba(229,228,226,0.60)", letterSpacing: "-0.01em" }}
              >
                {FINAL_CTA.body}
              </p>

              <div className="mt-10 flex flex-col sm:flex-row gap-4">
                <Link
                  href={FINAL_CTA.primary.href}
                  id="final-cta-primary"
                  className="inline-flex items-center justify-center h-12 px-8 text-sm font-semibold text-[#1C1412] transition-colors duration-200"
                  style={{ background: "#E5E4E2" }}
                  onMouseEnter={e => (e.currentTarget.style.background = "#C8B8AE")}
                  onMouseLeave={e => (e.currentTarget.style.background = "#E5E4E2")}
                >
                  {FINAL_CTA.primary.label}
                </Link>
                <Link
                  href={FINAL_CTA.secondary.href}
                  id="final-cta-secondary"
                  className="inline-flex items-center justify-center h-12 px-7 text-sm font-medium border transition-colors duration-200"
                  style={{
                    borderColor: "rgba(229,228,226,0.20)",
                    color: "rgba(229,228,226,0.70)",
                  }}
                  onMouseEnter={e => { e.currentTarget.style.borderColor = "rgba(229,228,226,0.50)"; e.currentTarget.style.color = "#E5E4E2"; }}
                  onMouseLeave={e => { e.currentTarget.style.borderColor = "rgba(229,228,226,0.20)"; e.currentTarget.style.color = "rgba(229,228,226,0.70)"; }}
                >
                  {FINAL_CTA.secondary.label}
                </Link>
              </div>
            </ScrollReveal>
          </div>
        </div>

        {/* Bottom rule */}
        <div
          className="mt-20 h-px"
          style={{ background: "rgba(229,228,226,0.08)" }}
          aria-hidden
        />
      </Container>
    </section>
  );
}
