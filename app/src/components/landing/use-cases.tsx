"use client";

import { useRef, useState } from "react";

import { Container, SectionHeading } from "@/components/landing/section";
import { USE_CASES } from "@/lib/landing-data";
import { cn } from "@/lib/utils";

export function UseCases() {
  const cases = USE_CASES.cases;
  const [activeId, setActiveId] = useState(cases[0]?.id ?? "");
  const tabRefs = useRef<Record<string, HTMLButtonElement | null>>({});

  const active = cases.find((item) => item.id === activeId) ?? cases[0];

  function onKeyDown(event: React.KeyboardEvent<HTMLButtonElement>, index: number) {
    const offsets: Record<string, number> = { ArrowRight: 1, ArrowLeft: -1 };
    const delta = offsets[event.key];
    if (delta === undefined) return;

    event.preventDefault();
    const next = cases[(index + delta + cases.length) % cases.length];
    if (!next) return;
    setActiveId(next.id);
    tabRefs.current[next.id]?.focus();
  }

  if (!active) return null;

  return (
    <section className="border-b">
      <Container className="py-20 sm:py-24 lg:py-28">
        <SectionHeading
          eyebrow={USE_CASES.eyebrow}
          title={USE_CASES.headline}
          body={USE_CASES.body}
        />

        <div className="mt-10 flex flex-wrap gap-2">
          {cases.map((item, index) => {
            const selected = item.id === active.id;
            return (
              <button
                key={item.id}
                type="button"
                role="tab"
                id={`usecase-tab-${item.id}`}
                aria-selected={selected}
                aria-controls="usecase-panel"
                tabIndex={selected ? 0 : -1}
                ref={(node) => {
                  tabRefs.current[item.id] = node;
                }}
                onClick={() => setActiveId(item.id)}
                onKeyDown={(event) => onKeyDown(event, index)}
                className={cn(
                  "rounded-lg border px-4 py-2 text-sm font-medium transition-colors",
                  "focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none",
                  selected
                    ? "border-primary/40 bg-primary/[0.04] text-primary"
                    : "text-muted-foreground hover:bg-muted/50 hover:text-foreground",
                )}
              >
                {item.label}
              </button>
            );
          })}
        </div>

        <div
          id="usecase-panel"
          role="tabpanel"
          aria-labelledby={`usecase-tab-${active.id}`}
          tabIndex={0}
          className="mt-8 grid gap-10 rounded-xl border bg-card p-6 focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none sm:p-8 lg:grid-cols-[1.4fr_1fr] lg:gap-14"
        >
          <div>
            <h3 className="text-xl font-semibold tracking-tight sm:text-2xl">
              {active.headline}
            </h3>
            <p className="mt-3 leading-relaxed text-muted-foreground text-pretty">
              {active.body}
            </p>
            <ul className="mt-6 space-y-2.5">
              {active.outcomes.map((outcome) => (
                <li key={outcome} className="flex gap-2.5 text-sm">
                  <span className="mt-2 size-1 shrink-0 rounded-full bg-primary" aria-hidden />
                  <span className="text-muted-foreground">{outcome}</span>
                </li>
              ))}
            </ul>
          </div>

          <dl className="grid content-start gap-px self-start overflow-hidden rounded-lg border bg-border">
            {active.surface.map((row) => (
              <div key={row.label} className="flex items-center justify-between gap-4 bg-card px-4 py-3">
                <dt className="text-xs text-muted-foreground">{row.label}</dt>
                <dd className="text-sm font-medium">{row.value}</dd>
              </div>
            ))}
          </dl>
        </div>
      </Container>
    </section>
  );
}
