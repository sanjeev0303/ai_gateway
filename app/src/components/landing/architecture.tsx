import { ScrollReveal } from "@/components/landing/scroll-reveal";
import { Container, SectionLabel } from "@/components/landing/section";
import { ARCHITECTURE } from "@/lib/landing-data";

export function Architecture() {
  return (
    <section
      className="relative overflow-hidden border-b border-[rgba(111,78,55,0.10)] grain"
      style={{ background: "var(--lavender)" }}
    >
      <Container className="py-24 sm:py-32 lg:py-40">
        {/* Z-pattern: steps left (5 cols), diagram right (7 cols) */}
        <div className="grid gap-16 lg:grid-cols-12 lg:gap-20 lg:items-start">

          {/* Left: timeline steps */}
          <div className="lg:col-span-5">
            <ScrollReveal>
              <SectionLabel>{ARCHITECTURE.eyebrow}</SectionLabel>
              <h2
                className="mt-4 font-display text-[clamp(2rem,4vw,3.5rem)] font-semibold text-[#1C1412] leading-[0.95]"
                style={{ fontFamily: "var(--font-cormorant)", letterSpacing: "-0.04em" }}
              >
                {ARCHITECTURE.headline}
              </h2>
              <p className="mt-6 text-base leading-relaxed text-[#4A3A30] max-w-[42ch]" style={{ letterSpacing: "-0.01em" }}>
                {ARCHITECTURE.body}
              </p>
            </ScrollReveal>

            {/* Vertical timeline */}
            <ScrollReveal delay={0.15} className="mt-12">
              <ol className="space-y-0">
                {ARCHITECTURE.steps.map((step, index) => (
                  <li key={step.label} className="flex gap-5 pb-8 last:pb-0">
                    {/* Line + dot */}
                    <div className="flex flex-col items-center shrink-0">
                      <div
                        className="size-5 rounded-full border-2 border-[#6F4E37] bg-lavender shrink-0 relative z-10"
                      />
                      {index < ARCHITECTURE.steps.length - 1 && (
                        <div
                          className="flex-1 w-px mt-2"
                          style={{ background: "rgba(111,78,55,0.20)" }}
                          aria-hidden
                        />
                      )}
                    </div>

                    <div className="pb-2">
                      <p
                        className="text-sm font-semibold text-[#1C1412]"
                        style={{ letterSpacing: "-0.02em" }}
                      >
                        {step.label}
                      </p>
                      <p className="mt-1.5 text-xs leading-relaxed text-[#4A3A30]">{step.detail}</p>
                    </div>
                  </li>
                ))}
              </ol>
            </ScrollReveal>
          </div>

          {/* Right: pipeline diagram + fields table */}
          <div className="lg:col-span-7 space-y-6">

            {/* Flow diagram */}
            <ScrollReveal delay={0.2}>
              <div
                className="border border-[rgba(111,78,55,0.16)] p-6"
                style={{ background: "rgba(229,228,226,0.60)" }}
              >
                <div className="flex items-center justify-between mb-6 pb-4 border-b border-[rgba(111,78,55,0.10)]">
                  <div className="flex items-center gap-2">
                    <div
                      className="size-1.5 rounded-full"
                      style={{ background: "#6F4E37" }}
                    />
                    <span
                      className="text-[10px] uppercase text-[#7A6458]"
                      style={{ fontFamily: "var(--font-fira-code)", letterSpacing: "0.12em" }}
                    >
                      Flow Architecture
                    </span>
                  </div>
                  <span
                    className="text-[10px] text-[#6F4E37] border border-[rgba(111,78,55,0.25)] px-2.5 py-1"
                    style={{ fontFamily: "var(--font-fira-code)" }}
                  >
                    Sub-millisecond overhead
                  </span>
                </div>

                <div className="flex flex-col md:flex-row items-center justify-between gap-3 text-center">
                  {[
                    { label: "App / Client", sub: "OpenAI SDK" },
                    { label: "Sentinel Core", sub: "Policy & Budget", highlight: true },
                    { label: "LLM Provider", sub: "Anthropic / OpenAI" },
                  ].map((node, i) => (
                    <div key={node.label} className="contents">
                      <div
                        className="w-full md:flex-1 border p-4"
                        style={{
                          borderColor: node.highlight ? "#6F4E37" : "rgba(111,78,55,0.18)",
                          background: node.highlight ? "rgba(111,78,55,0.08)" : "rgba(229,228,226,0.8)",
                        }}
                      >
                        <p
                          className="text-xs font-semibold"
                          style={{
                            fontFamily: "var(--font-fira-code)",
                            color: node.highlight ? "#6F4E37" : "#1C1412",
                          }}
                        >
                          {node.label}
                        </p>
                        <p
                          className="text-[10px] mt-1"
                          style={{ color: "rgba(74,58,48,0.70)" }}
                        >
                          {node.sub}
                        </p>
                      </div>

                      {i < 2 && (
                        <div className="hidden md:flex items-center px-2" aria-hidden>
                          <svg className="w-10 h-3" viewBox="0 0 40 12">
                            <line
                              x1="0" y1="6" x2="40" y2="6"
                              stroke="#6F4E37"
                              strokeWidth="1"
                              strokeDasharray="3 2"
                              className="animate-dash"
                            />
                          </svg>
                        </div>
                      )}
                    </div>
                  ))}
                </div>
              </div>
            </ScrollReveal>

            {/* Attribution payload fields */}
            <ScrollReveal delay={0.28}>
              <div
                className="border border-[rgba(111,78,55,0.16)] overflow-hidden"
                style={{ background: "rgba(229,228,226,0.60)" }}
              >
                <div className="px-6 py-4 border-b border-[rgba(111,78,55,0.10)]">
                  <h3 className="text-sm font-semibold text-[#1C1412]" style={{ letterSpacing: "-0.02em" }}>
                    Attribution & Telemetry Payload
                  </h3>
                  <p className="mt-1 text-xs text-[#7A6458]">
                    Deterministic audit headers injected on each request trace.
                  </p>
                </div>
                <dl className="divide-y divide-[rgba(111,78,55,0.07)]">
                  {ARCHITECTURE.fields.map((row) => (
                    <div
                      key={row.field}
                      className="grid gap-3 py-3 px-6 sm:grid-cols-[160px_1fr] sm:gap-6 items-center"
                    >
                      <dt>
                        <span
                          className="text-xs font-semibold text-[#6F4E37] border border-[rgba(111,78,55,0.20)] px-2 py-0.5"
                          style={{ fontFamily: "var(--font-fira-code)" }}
                        >
                          {row.field}
                        </span>
                      </dt>
                      <dd className="text-xs leading-relaxed text-[#4A3A30]">{row.detail}</dd>
                    </div>
                  ))}
                </dl>
              </div>
            </ScrollReveal>
          </div>
        </div>
      </Container>
    </section>
  );
}
