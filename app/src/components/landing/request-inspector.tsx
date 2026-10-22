"use client";

import { useRef, useState } from "react";
import { ScrollReveal } from "@/components/landing/scroll-reveal";
import { Container, SectionLabel } from "@/components/landing/section";
import { REQUEST_INSPECTOR } from "@/lib/landing-data";
import { cn } from "@/lib/utils";

function formatCost(cost: number) {
  return `$${cost.toFixed(4)}`;
}
function formatTokens(tokens: number) {
  return tokens.toLocaleString("en-US");
}

export function RequestInspector() {
  const samples = REQUEST_INSPECTOR.samples;
  const [activeId, setActiveId] = useState(samples[0]?.id ?? "");
  const tabRefs = useRef<Record<string, HTMLButtonElement | null>>({});
  const active = samples.find((s) => s.id === activeId) ?? samples[0];

  function onKeyDown(event: React.KeyboardEvent<HTMLButtonElement>, index: number) {
    const offsets: Record<string, number> = { ArrowDown: 1, ArrowRight: 1, ArrowUp: -1, ArrowLeft: -1 };
    const delta = offsets[event.key];
    if (delta === undefined) return;
    event.preventDefault();
    const next = samples[(index + delta + samples.length) % samples.length];
    if (!next) return;
    setActiveId(next.id);
    tabRefs.current[next.id]?.focus();
  }

  if (!active) return null;

  return (
    <section
      className="relative overflow-hidden border-b border-[rgba(111,78,55,0.10)] weave"
      style={{ background: "var(--canvas-warm)" }}
    >
      <Container className="py-24 sm:py-32 lg:py-40">
        {/* Z-reverse: text right (5 cols), inspector left (7 cols) */}
        <div className="grid gap-16 lg:grid-cols-12 lg:gap-20 lg:items-start">

          {/* Left: interactive inspector */}
          <ScrollReveal className="lg:col-span-7" delay={0.12}>
            <div className="border border-[rgba(111,78,55,0.16)] overflow-hidden">
              {/* Tab strip */}
              <div
                role="tablist"
                aria-label="Sample requests"
                aria-orientation="vertical"
                className="border-b border-[rgba(111,78,55,0.10)] grid grid-cols-2 sm:grid-cols-4"
              >
                {samples.map((sample, index) => {
                  const selected = sample.id === active.id;
                  return (
                    <button
                      key={sample.id}
                      type="button"
                      role="tab"
                      id={`sample-tab-${sample.id}`}
                      aria-selected={selected}
                      aria-controls="sample-panel"
                      tabIndex={selected ? 0 : -1}
                      ref={(node) => { tabRefs.current[sample.id] = node; }}
                      onClick={() => setActiveId(sample.id)}
                      onKeyDown={(e) => onKeyDown(e, index)}
                      className={cn(
                        "px-4 py-3 text-left border-r border-[rgba(111,78,55,0.10)] last:border-r-0 transition-colors duration-150",
                        "focus-visible:outline-2 focus-visible:outline-[#6F4E37]",
                        selected
                          ? "bg-[#6F4E37] text-[#F5F3F0]"
                          : "bg-transparent text-[#7A6458]"
                      )}
                    >
                      <span
                        className="block text-[10px] font-medium mb-0.5"
                        style={{ fontFamily: "var(--font-fira-code)" }}
                      >
                        {sample.id}
                      </span>
                      <span className="block text-xs font-semibold truncate" style={{ letterSpacing: "-0.02em" }}>
                        {sample.model}
                      </span>
                    </button>
                  );
                })}
              </div>

              {/* Detail panel */}
              <div
                id="sample-panel"
                role="tabpanel"
                aria-labelledby={`sample-tab-${active.id}`}
                tabIndex={0}
                className="p-6 sm:p-8 focus-visible:outline-none"
                style={{ background: "rgba(245,243,240,0.5)" }}
              >
                {/* Header */}
                <div className="flex items-start justify-between gap-4 pb-5 border-b border-[rgba(111,78,55,0.10)] mb-5">
                  <div>
                    <p className="text-lg font-semibold text-[#1C1412]" style={{ fontFamily: "var(--font-cormorant)", letterSpacing: "-0.03em" }}>
                      {active.model}
                    </p>
                    <p className="text-xs text-[#7A6458] mt-0.5" style={{ fontFamily: "var(--font-fira-code)" }}>
                      via {active.provider}
                    </p>
                  </div>
                  <span
                    className={cn(
                      "text-[10px] font-semibold px-2 py-1 border",
                      active.outcome === "Error"
                        ? "border-[#B54B2A] text-[#B54B2A] bg-[rgba(181,75,42,0.06)]"
                        : "border-[#4A7A5A] text-[#4A7A5A] bg-[rgba(74,122,90,0.06)]"
                    )}
                    style={{ fontFamily: "var(--font-fira-code)" }}
                  >
                    {active.outcome}
                  </span>
                </div>

                {/* Fields */}
                <dl className="space-y-0">
                  {[
                    { label: "Project", value: active.project },
                    { label: "Team", value: active.team },
                    { label: "Status", value: active.status },
                    { label: "Timestamp", value: active.time },
                    { label: "Tokens", value: formatTokens(active.tokens) },
                    { label: "Attributed cost", value: formatCost(active.cost) },
                  ].map(({ label, value }) => (
                    <div
                      key={label}
                      className="grid grid-cols-[120px_1fr] gap-4 py-2.5 border-b border-[rgba(111,78,55,0.07)] last:border-b-0"
                    >
                      <dt className="text-xs text-[#7A6458]" style={{ fontFamily: "var(--font-fira-code)" }}>
                        {label}
                      </dt>
                      <dd className="text-sm font-semibold text-[#1C1412] tabular-nums" style={{ letterSpacing: "-0.01em" }}>
                        {value}
                      </dd>
                    </div>
                  ))}
                </dl>

                {/* Trace note */}
                <div
                  className="mt-5 p-4 border-l-2 border-[#6F4E37]"
                  style={{ background: "rgba(111,78,55,0.04)" }}
                >
                  <p className="text-[10px] uppercase tracking-label text-[#6F4E37] mb-1" style={{ fontFamily: "var(--font-fira-code)", letterSpacing: "0.12em" }}>
                    Evaluation trace
                  </p>
                  <p className="text-xs leading-relaxed text-[#4A3A30]">{active.detail}</p>
                </div>
              </div>
            </div>

            <p className="mt-4 text-xs text-[#7A6458]" style={{ fontFamily: "var(--font-fira-code)" }}>
              {REQUEST_INSPECTOR.note}
            </p>
          </ScrollReveal>

          {/* Right: text context — sticky */}
          <div className="lg:col-span-5 lg:sticky lg:top-32">
            <ScrollReveal>
              <SectionLabel>{REQUEST_INSPECTOR.eyebrow}</SectionLabel>
              <h2
                className="mt-4 font-display text-[clamp(2rem,4vw,3.5rem)] font-semibold text-[#1C1412] leading-[0.95]"
                style={{ fontFamily: "var(--font-cormorant)", letterSpacing: "-0.04em" }}
              >
                {REQUEST_INSPECTOR.headline}
              </h2>
              <p className="mt-6 text-base leading-relaxed text-[#4A3A30] max-w-[44ch]" style={{ letterSpacing: "-0.01em" }}>
                {REQUEST_INSPECTOR.body}
              </p>

              {/* Feature bullets — clean list style */}
              <ul className="mt-8 space-y-4">
                {[
                  "Model, provider, tokens recorded per request",
                  "Outcome and status code preserved",
                  "Cost derived at write time",
                  "Rolls up to project and team",
                ].map((item) => (
                  <li key={item} className="flex items-start gap-3">
                    <div
                      className="mt-1.5 h-px w-4 shrink-0"
                      style={{ background: "#6F4E37" }}
                      aria-hidden
                    />
                    <span className="text-sm text-[#4A3A30]" style={{ letterSpacing: "-0.01em" }}>{item}</span>
                  </li>
                ))}
              </ul>
            </ScrollReveal>
          </div>
        </div>
      </Container>
    </section>
  );
}
