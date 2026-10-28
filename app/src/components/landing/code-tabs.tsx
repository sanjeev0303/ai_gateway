"use client";

import { useEffect, useRef, useState } from "react";
import { ScrollReveal } from "@/components/landing/scroll-reveal";
import { Container, SectionLabel } from "@/components/landing/section";
import { CODE_TABS } from "@/lib/landing-data";
import { cn } from "@/lib/utils";

export function CodeTabs() {
  const samples = CODE_TABS.samples;
  const [activeId, setActiveId] = useState(samples[0]?.id ?? "");
  const [copied, setCopied] = useState(false);
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const tabRefs = useRef<Record<string, HTMLButtonElement | null>>({});
  const active = samples.find((s) => s.id === activeId) ?? samples[0];

  useEffect(() => () => { if (timer.current) clearTimeout(timer.current); }, []);

  if (!active) return null;

  async function copy() {
    if (!active) return;
    try {
      await navigator.clipboard.writeText(active.code);
      setCopied(true);
      if (timer.current) clearTimeout(timer.current);
      timer.current = setTimeout(() => setCopied(false), 2000);
    } catch { setCopied(false); }
  }

  function onKeyDown(event: React.KeyboardEvent<HTMLButtonElement>, index: number) {
    const offsets: Record<string, number> = { ArrowRight: 1, ArrowLeft: -1 };
    const delta = offsets[event.key];
    if (delta === undefined) return;
    event.preventDefault();
    const next = samples[(index + delta + samples.length) % samples.length];
    if (!next) return;
    setActiveId(next.id);
    tabRefs.current[next.id]?.focus();
  }

  return (
    <section
      id="integration"
      className="relative overflow-hidden border-b border-[rgba(111,78,55,0.10)] weave scroll-mt-14"
      style={{ background: "var(--canvas)" }}
    >
      <Container className="py-24 sm:py-32 lg:py-40">
        {/* Z-pattern: heading left (5 cols), code right (7 cols) */}
        <div className="grid gap-16 lg:grid-cols-12 lg:gap-20 lg:items-start">

          {/* Left: editorial heading */}
          <div className="lg:col-span-5 lg:sticky lg:top-32">
            <ScrollReveal>
              <SectionLabel>{CODE_TABS.eyebrow}</SectionLabel>
              <h2
                className="mt-4 font-display text-[clamp(2rem,4vw,3.5rem)] font-semibold text-[#1C1412] leading-[0.95]"
                style={{ fontFamily: "var(--font-cormorant)", letterSpacing: "-0.04em" }}
              >
                {CODE_TABS.headline}
              </h2>
              <p className="mt-6 text-base leading-relaxed text-[#4A3A30] max-w-[42ch]" style={{ letterSpacing: "-0.01em" }}>
                {CODE_TABS.body}
              </p>

              <ul className="mt-8 space-y-3">
                {[
                  "Any HTTP client — no wrapper SDK required",
                  "One environment variable to configure",
                  "Anthropic-format messages, forwarded as-is",
                ].map((item) => (
                  <li key={item} className="flex items-start gap-3 text-sm text-[#4A3A30]">
                    <div className="mt-2 h-px w-4 shrink-0" style={{ background: "#6F4E37" }} aria-hidden />
                    {item}
                  </li>
                ))}
              </ul>

              <p className="mt-8 text-xs text-[#7A6458]" style={{ fontFamily: "var(--font-fira-code)" }}>
                {CODE_TABS.note}
              </p>
            </ScrollReveal>
          </div>

          {/* Right: code panel */}
          <ScrollReveal className="lg:col-span-7" delay={0.15}>
            <div className="overflow-hidden border border-[rgba(111,78,55,0.16)]" style={{ background: "#1C1412" }}>
              {/* Code topbar */}
              <div
                className="flex items-center justify-between px-5 py-3 border-b"
                style={{ borderColor: "rgba(229,228,226,0.08)" }}
              >
                {/* Language tabs */}
                <div role="tablist" aria-label="Language" className="flex gap-0">
                  {samples.map((sample, index) => {
                    const selected = sample.id === active.id;
                    return (
                      <button
                        key={sample.id}
                        type="button"
                        role="tab"
                        aria-selected={selected}
                        aria-controls="code-panel"
                        tabIndex={selected ? 0 : -1}
                        ref={(node) => { tabRefs.current[sample.id] = node; }}
                        onClick={() => setActiveId(sample.id)}
                        onKeyDown={(e) => onKeyDown(e, index)}
                        className={cn(
                          "px-4 py-1.5 text-xs font-semibold border-r border-[rgba(229,228,226,0.08)] last:border-r-0 transition-colors duration-150",
                          "focus-visible:outline-2 focus-visible:outline-[#6F4E37]",
                          selected ? "text-[#E5E4E2] bg-[rgba(229,228,226,0.06)]" : "text-[rgba(229,228,226,0.40)]"
                        )}
                        style={{ fontFamily: "var(--font-fira-code)" }}
                      >
                        {sample.label}
                      </button>
                    );
                  })}
                </div>

                {/* Copy button */}
                <button
                  type="button"
                  onClick={copy}
                  className="text-xs font-medium transition-colors duration-150 focus-visible:outline-2 focus-visible:outline-[#6F4E37] px-3.5 py-1.5 border border-[rgba(229,228,226,0.12)] text-[rgba(229,228,226,0.70)] hover:text-[#E5E4E2]"
                  style={{ fontFamily: "var(--font-fira-code)" }}
                >
                  {copied ? "Copied" : "Copy"}
                </button>
              </div>

              {/* Code content */}
              <div
                id="code-panel"
                role="tabpanel"
                aria-label={`${active.label} example`}
                className="overflow-x-auto p-6"
              >
                <pre className="text-sm leading-relaxed text-[rgba(229,228,226,0.85)]" style={{ fontFamily: "var(--font-fira-code)" }}>
                  <code>{active.code}</code>
                </pre>
              </div>
            </div>

            <p aria-live="polite" className="sr-only">
              {copied ? `${active.label} snippet copied to clipboard` : ""}
            </p>
          </ScrollReveal>
        </div>
      </Container>
    </section>
  );
}
