"use client";

import { ScrollReveal } from "@/components/landing/scroll-reveal";
import { Container, SectionLabel } from "@/components/landing/section";
import { GATEWAY_FLOW } from "@/lib/landing-data";

export function GatewayFlow() {
  const { inbound, outbound, loop } = GATEWAY_FLOW;

  return (
    <section
      className="relative overflow-hidden border-b border-[rgba(111,78,55,0.10)] grain"
      style={{ background: "var(--lavender)" }}
    >
      <Container className="py-24 sm:py-32 lg:py-40">

        {/* Centered heading block — rule of thirds center axis */}
        <ScrollReveal>
          <div className="max-w-2xl">
            <SectionLabel>{GATEWAY_FLOW.eyebrow}</SectionLabel>
            <h2
              className="mt-4 font-display text-[clamp(2.5rem,5vw,4rem)] font-semibold text-[#1C1412] leading-[0.95]"
              style={{ fontFamily: "var(--font-cormorant)", letterSpacing: "-0.04em" }}
            >
              {GATEWAY_FLOW.headline}
            </h2>
            <p className="mt-6 text-base leading-relaxed text-[#4A3A30] max-w-[52ch]" style={{ letterSpacing: "-0.01em" }}>
              {GATEWAY_FLOW.body}
            </p>
          </div>
        </ScrollReveal>

        {/* Pipeline visual — horizontal flow with connectors */}
        <ScrollReveal delay={0.15} className="mt-16">
          <div className="relative">
            {/* Connector line behind nodes */}
            <div
              className="absolute inset-y-0 left-0 right-0 top-1/2 hidden md:block"
              style={{
                height: "1px",
                top: "50%",
                background: "linear-gradient(90deg, transparent, rgba(111,78,55,0.22) 20%, rgba(111,78,55,0.22) 80%, transparent)",
              }}
              aria-hidden
            />

            <ol
              className="relative grid grid-cols-1 gap-6 md:grid-cols-[1fr_auto_1fr_auto_1fr]"
              aria-label="Request pipeline"
            >
              {inbound.map((step, index) => (
                <div key={step.label} className="contents">
                  <li className="relative rounded-none border border-[rgba(111,78,55,0.16)] bg-[rgba(229,228,226,0.8)] p-6 backdrop-blur-sm">
                    <p
                      className="font-mono text-[10px] text-[#7A6458] mb-3"
                      style={{ fontFamily: "var(--font-fira-code)", letterSpacing: "0.10em" }}
                    >
                      {String(index + 1).padStart(2, "0")}
                    </p>
                    <p className="text-base font-semibold text-[#1C1412]" style={{ letterSpacing: "-0.02em" }}>
                      {step.label}
                    </p>
                    <p className="mt-1.5 text-xs leading-relaxed text-[#7A6458]">{step.detail}</p>
                  </li>

                  {/* Connector dot */}
                  <li className="hidden md:flex items-center justify-center" aria-hidden>
                    <div
                      className="size-2 rounded-full"
                      style={{ background: "#6F4E37" }}
                    />
                  </li>
                </div>
              ))}

              {/* Outbound node — highlighted */}
              <li className="relative border border-[#6F4E37] bg-[#6F4E37] p-6">
                <p
                  className="font-mono text-[10px] text-[rgba(245,243,240,0.6)] mb-3"
                  style={{ fontFamily: "var(--font-fira-code)", letterSpacing: "0.10em" }}
                >
                  03
                </p>
                <p className="text-base font-semibold text-[#F5F3F0]" style={{ letterSpacing: "-0.02em" }}>
                  {outbound.label}
                </p>
                <p className="mt-1.5 text-xs leading-relaxed text-[rgba(245,243,240,0.7)]">{outbound.detail}</p>
              </li>
            </ol>
          </div>
        </ScrollReveal>

        {/* Continuous pipeline loop — 4 steps in a row */}
        <ScrollReveal delay={0.25} className="mt-20">
          <div
            className="border-t border-[rgba(111,78,55,0.12)] pt-12"
          >
            <p
              className="font-mono text-[10px] tracking-label text-[#7A6458] uppercase mb-8"
              style={{ fontFamily: "var(--font-fira-code)", letterSpacing: "0.14em" }}
            >
              Continuous pipeline — executed on every request
            </p>

            <ol className="grid gap-8 sm:grid-cols-2 lg:grid-cols-4">
              {loop.map((step, index) => (
                <li key={step.title} className="flex gap-4 items-start">
                  {/* Number */}
                  <span
                    className="shrink-0 font-mono text-3xl font-semibold text-[rgba(111,78,55,0.18)] leading-none tabular-nums"
                    style={{ fontFamily: "var(--font-cormorant)", letterSpacing: "-0.04em" }}
                    aria-hidden
                  >
                    {String(index + 1).padStart(2, "0")}
                  </span>
                  <div>
                    <h3 className="text-sm font-semibold text-[#1C1412]" style={{ letterSpacing: "-0.02em" }}>
                      {step.title}
                    </h3>
                    <p className="mt-1.5 text-xs leading-relaxed text-[#4A3A30]">{step.detail}</p>
                  </div>
                </li>
              ))}
            </ol>
          </div>
        </ScrollReveal>
      </Container>
    </section>
  );
}
