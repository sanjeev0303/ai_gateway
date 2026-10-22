"use client";

import Link from "next/link";
import { ScrollReveal } from "@/components/landing/scroll-reveal";
import { Container, SectionLabel } from "@/components/landing/section";
import { PRICING_TEASER } from "@/lib/landing-data";

const DEPLOYMENT_OPTIONS = [
  { label: "Cloud", sub: "Managed, always up-to-date" },
  { label: "VPC", sub: "Your cloud, our control plane" },
  { label: "On-premise", sub: "Air-gapped enterprise deployment" },
] as const;

export function PricingTeaser() {
  const { eyebrow, headline, body, points, primary, secondary } = PRICING_TEASER;

  return (
    <section
      id="pricing"
      className="relative overflow-hidden border-b border-[rgba(111,78,55,0.10)] weave grain scroll-mt-14"
      style={{ background: "var(--canvas-warm)" }}
    >
      <Container className="py-24 sm:py-32 lg:py-40">

        {/* Z-pattern: pricing list left (7), CTA right (5) */}
        <div className="grid gap-16 lg:grid-cols-12 lg:gap-20 lg:items-start">

          {/* Left: heading + points */}
          <div className="lg:col-span-7">
            <ScrollReveal>
              <SectionLabel>{eyebrow}</SectionLabel>
              <h2
                className="mt-4 font-display text-[clamp(2.5rem,5vw,4rem)] font-semibold text-[#1C1412] leading-[0.95]"
                style={{ fontFamily: "var(--font-cormorant)", letterSpacing: "-0.04em" }}
              >
                {headline}
              </h2>
              <p className="mt-6 text-base leading-relaxed text-[#4A3A30] max-w-[48ch]" style={{ letterSpacing: "-0.01em" }}>
                {body}
              </p>
            </ScrollReveal>

            {/* Points: editorial list */}
            <ul className="mt-10 space-y-0 border-t border-[rgba(111,78,55,0.10)]">
              {points.map((point, index) => (
                <li key={point} className="border-b border-[rgba(111,78,55,0.10)]">
                  <ScrollReveal delay={index * 0.08}>
                    <div className="py-5 flex items-start gap-5">
                      <span
                        className="font-mono text-[10px] text-[#7A6458] shrink-0 pt-0.5"
                        style={{ fontFamily: "var(--font-fira-code)" }}
                      >
                        {String(index + 1).padStart(2, "0")}
                      </span>
                      <span className="text-sm font-medium text-[#1C1412]" style={{ letterSpacing: "-0.01em" }}>
                        {point}
                      </span>
                    </div>
                  </ScrollReveal>
                </li>
              ))}
            </ul>

            {/* Deployment options */}
            <ScrollReveal delay={0.25} className="mt-14">
              <p
                className="font-mono text-[10px] uppercase tracking-label text-[#7A6458] mb-5"
                style={{ fontFamily: "var(--font-fira-code)", letterSpacing: "0.14em" }}
              >
                Deployment options
              </p>
              <div className="grid grid-cols-3 gap-0 border border-[rgba(111,78,55,0.16)]">
                {DEPLOYMENT_OPTIONS.map((opt, i) => (
                  <div
                    key={opt.label}
                    className="p-5 border-r border-[rgba(111,78,55,0.10)] last:border-r-0"
                  >
                    <p className="text-sm font-semibold text-[#1C1412]" style={{ letterSpacing: "-0.02em" }}>
                      {opt.label}
                    </p>
                    <p className="mt-1 text-xs text-[#7A6458]">{opt.sub}</p>
                  </div>
                ))}
              </div>
            </ScrollReveal>
          </div>

          {/* Right: CTA card */}
          <div className="lg:col-span-5">
            <ScrollReveal delay={0.2}>
              <div
                className="border border-[#6F4E37] p-8"
                style={{ background: "rgba(111,78,55,0.04)" }}
              >
                <p
                  className="text-[10px] uppercase text-[#6F4E37]"
                  style={{ fontFamily: "var(--font-fira-code)", letterSpacing: "0.14em" }}
                >
                  Predictable enterprise tiering
                </p>
                <h3
                  className="mt-3 text-2xl font-semibold text-[#1C1412]"
                  style={{ fontFamily: "var(--font-cormorant)", letterSpacing: "-0.03em" }}
                >
                  Ready to deploy Sentinel?
                </h3>
                <p className="mt-3 text-sm leading-relaxed text-[#4A3A30]">{body}</p>

                <div className="mt-8 flex flex-col gap-3">
                  <Link
                    href={primary.href}
                    id="pricing-cta-primary"
                    className="flex items-center justify-center h-12 px-6 text-sm font-semibold text-[#F5F3F0] transition-colors duration-200"
                    style={{ background: "#6F4E37" }}
                    onMouseEnter={e => (e.currentTarget.style.background = "#4A3224")}
                    onMouseLeave={e => (e.currentTarget.style.background = "#6F4E37")}
                  >
                    {primary.label}
                  </Link>
                  <Link
                    href={secondary.href}
                    id="pricing-cta-secondary"
                    className="flex items-center justify-center h-12 px-6 text-sm font-medium text-[#6F4E37] border border-[rgba(111,78,55,0.28)] transition-colors duration-200"
                  >
                    {secondary.label}
                  </Link>
                </div>
              </div>
            </ScrollReveal>
          </div>
        </div>
      </Container>
    </section>
  );
}
