"use client";

import { useState } from "react";
import { ScrollReveal } from "@/components/landing/scroll-reveal";
import { Container, SectionLabel } from "@/components/landing/section";
import { ATTRIBUTION, type AttributionMetric } from "@/lib/landing-data";
import { cn } from "@/lib/utils";

export function Attribution() {
  const metrics = ATTRIBUTION.metrics;
  const [activeId, setActiveId] = useState<AttributionMetric>("spend");
  const active = metrics.find((m) => m.id === activeId) ?? metrics[0];

  if (!active) return null;

  const values = ATTRIBUTION.rows.map((row) => row[active.id]);
  const max = Math.max(...values, 1);

  return (
    <section
      className="relative overflow-hidden border-b border-[rgba(111,78,55,0.10)] weave grain"
      style={{ background: "var(--canvas-warm)" }}
    >
      <Container className="py-24 sm:py-32 lg:py-40">
        {/* Z-layout: heading left, chart right */}
        <div className="grid gap-16 lg:grid-cols-12 lg:gap-20 lg:items-start">

          {/* Left: heading + controls */}
          <div className="lg:col-span-4 lg:sticky lg:top-32">
            <ScrollReveal>
              <SectionLabel>{ATTRIBUTION.eyebrow}</SectionLabel>
              <h2
                className="mt-4 font-display text-[clamp(2rem,4vw,3.5rem)] font-semibold text-[#1C1412] leading-[0.95]"
                style={{ fontFamily: "var(--font-cormorant)", letterSpacing: "-0.04em" }}
              >
                {ATTRIBUTION.headline}
              </h2>
              <p className="mt-6 text-base leading-relaxed text-[#4A3A30] max-w-[36ch]" style={{ letterSpacing: "-0.01em" }}>
                {ATTRIBUTION.body}
              </p>

              {/* Metric toggle — underline tab style */}
              <div
                role="group"
                aria-label="Attribution metric"
                className="mt-10 flex border-b border-[rgba(111,78,55,0.12)]"
              >
                {metrics.map((metric) => {
                  const selected = metric.id === active.id;
                  return (
                    <button
                      key={metric.id}
                      type="button"
                      aria-pressed={selected}
                      onClick={() => setActiveId(metric.id)}
                      className={cn(
                        "relative px-4 py-2.5 text-xs font-semibold transition-colors duration-150 focus-visible:outline-2 focus-visible:outline-[#6F4E37]",
                        selected ? "text-[#6F4E37]" : "text-[#7A6458]"
                      )}
                      style={{ fontFamily: "var(--font-fira-code)" }}
                    >
                      {metric.label}
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
              <p className="mt-3 text-[11px] text-[#7A6458]" style={{ fontFamily: "var(--font-fira-code)" }}>
                {active.caption}
              </p>
            </ScrollReveal>
          </div>

          {/* Right: horizontal bar chart */}
          <ScrollReveal className="lg:col-span-8" delay={0.15}>
            <div>
              {/* Column headers */}
              <div className="flex items-baseline justify-between mb-4 pb-3 border-b border-[rgba(111,78,55,0.12)]">
                <span className="text-[10px] font-semibold text-[#7A6458] uppercase" style={{ fontFamily: "var(--font-fira-code)", letterSpacing: "0.10em" }}>
                  Team
                </span>
                <span className="text-[10px] font-semibold text-[#7A6458] uppercase" style={{ fontFamily: "var(--font-fira-code)", letterSpacing: "0.10em" }}>
                  {active.label} ({active.unit})
                </span>
              </div>

              <ul className="space-y-6">
                {ATTRIBUTION.rows.map((row) => {
                  const value = row[active.id];
                  const width = Math.max((value / max) * 100, 2);
                  return (
                    <li key={row.label}>
                      <div className="flex items-baseline justify-between gap-4 mb-2">
                        <span
                          className="text-sm font-semibold text-[#1C1412] truncate"
                          style={{ letterSpacing: "-0.02em" }}
                        >
                          {row.label}
                        </span>
                        <span
                          className="shrink-0 text-sm font-semibold tabular-nums text-[#6F4E37]"
                          style={{ fontFamily: "var(--font-cormorant)", letterSpacing: "-0.02em", fontSize: "1.1rem" }}
                        >
                          {active.format(value)}
                        </span>
                      </div>
                      {/* Bar */}
                      <div
                        aria-hidden
                        className="h-1.5 w-full overflow-hidden"
                        style={{ background: "rgba(111,78,55,0.10)" }}
                      >
                        <div
                          className="h-full transition-all duration-700 ease-out"
                          style={{ width: `${width}%`, background: "#6F4E37" }}
                        />
                      </div>
                    </li>
                  );
                })}
              </ul>

              <p className="mt-8 text-[10px] text-[#7A6458]" style={{ fontFamily: "var(--font-fira-code)" }}>
                {ATTRIBUTION.note}
              </p>
            </div>

            {/* Screen reader accessible table */}
            <table className="sr-only">
              <caption>{`${active.label} by team`}</caption>
              <thead>
                <tr>
                  <th scope="col">Team</th>
                  <th scope="col">{active.label}</th>
                </tr>
              </thead>
              <tbody>
                {ATTRIBUTION.rows.map((row) => (
                  <tr key={row.label}>
                    <th scope="row">{row.label}</th>
                    <td>{active.format(row[active.id])}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </ScrollReveal>
        </div>
      </Container>
    </section>
  );
}
