import { ScrollReveal } from "@/components/landing/scroll-reveal";
import { Container, SectionLabel } from "@/components/landing/section";
import { TRUST } from "@/lib/landing-data";

export function Trust() {
  return (
    <section
      id="trust"
      className="relative overflow-hidden border-b border-[rgba(111,78,55,0.10)] grain scroll-mt-14"
      style={{ background: "var(--canvas)" }}
    >
      <Container className="py-24 sm:py-32 lg:py-40">
        {/* Centered max-width heading */}
        <ScrollReveal>
          <div className="max-w-xl">
            <SectionLabel>{TRUST.eyebrow}</SectionLabel>
            <h2
              className="mt-4 font-display text-[clamp(2.5rem,5vw,4rem)] font-semibold text-[#1C1412] leading-[0.95]"
              style={{ fontFamily: "var(--font-cormorant)", letterSpacing: "-0.04em" }}
            >
              {TRUST.headline}
            </h2>
            <p className="mt-6 text-base leading-relaxed text-[#4A3A30] max-w-[48ch]" style={{ letterSpacing: "-0.01em" }}>
              {TRUST.body}
            </p>
          </div>
        </ScrollReveal>

        {/* 2-column editorial list — no cards */}
        <ul className="mt-20 grid gap-0 md:grid-cols-2 border-t border-l border-[rgba(111,78,55,0.10)]">
          {TRUST.items.map((mechanism, index) => (
            <li
              key={mechanism.title}
              className="border-b border-r border-[rgba(111,78,55,0.10)] p-8 sm:p-10"
            >
              <ScrollReveal delay={index * 0.09}>
                {/* Accent line top */}
                <div
                  className="h-px mb-6 w-10"
                  style={{ background: "#6F4E37" }}
                  aria-hidden
                />
                <h3
                  className="text-xl font-semibold text-[#1C1412]"
                  style={{ fontFamily: "var(--font-dm-sans)", letterSpacing: "-0.02em" }}
                >
                  {mechanism.title}
                </h3>
                <p className="mt-3 text-sm leading-relaxed text-[#4A3A30] max-w-[38ch]">
                  {mechanism.body}
                </p>
              </ScrollReveal>
            </li>
          ))}
        </ul>
      </Container>
    </section>
  );
}
