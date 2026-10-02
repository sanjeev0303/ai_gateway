"use client";

import { useRef, useState } from "react";

import { Container, Note, SectionHeading } from "@/components/landing/section";
import { REQUEST_INSPECTOR } from "@/lib/landing-data";
import { cn } from "@/lib/utils";

function formatCost(cost: number) {
  return `$${cost.toFixed(4)}`;
}

function formatTokens(tokens: number) {
  return tokens.toLocaleString("en-US");
}

function DetailRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="grid grid-cols-[104px_1fr] gap-3 py-2">
      <dt className="text-xs text-muted-foreground">{label}</dt>
      <dd className="min-w-0 text-sm tabular-nums">{value}</dd>
    </div>
  );
}

export function RequestInspector() {
  const samples = REQUEST_INSPECTOR.samples;
  const [activeId, setActiveId] = useState(samples[0]?.id ?? "");
  const tabRefs = useRef<Record<string, HTMLButtonElement | null>>({});
  const active = samples.find((sample) => sample.id === activeId) ?? samples[0];

  // Vertical at lg+, horizontal scroller below — so both key axes move between samples.
  function onKeyDown(event: React.KeyboardEvent<HTMLButtonElement>, index: number) {
    const offsets: Record<string, number> = {
      ArrowDown: 1,
      ArrowRight: 1,
      ArrowUp: -1,
      ArrowLeft: -1,
    };
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
    <section className="border-b">
      <Container className="py-20 sm:py-24 lg:py-28">
        <SectionHeading
          eyebrow={REQUEST_INSPECTOR.eyebrow}
          title={REQUEST_INSPECTOR.headline}
          body={REQUEST_INSPECTOR.body}
        />

        <div className="mt-12 grid gap-4 lg:grid-cols-[minmax(0,340px)_1fr]">
          <div
            role="tablist"
            aria-label="Sample requests"
            aria-orientation="vertical"
            className="flex gap-2 overflow-x-auto pb-1 lg:flex-col lg:overflow-visible lg:pb-0"
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
                  ref={(node) => {
                    tabRefs.current[sample.id] = node;
                  }}
                  onClick={() => setActiveId(sample.id)}
                  onKeyDown={(event) => onKeyDown(event, index)}
                  className={cn(
                    "min-w-56 shrink-0 rounded-lg border p-3 text-left transition-colors lg:min-w-0",
                    "focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none",
                    selected
                      ? "border-primary/40 bg-primary/[0.04]"
                      : "bg-card hover:bg-muted/50",
                  )}
                >
                  <span className="flex items-center justify-between gap-2">
                    <span className="font-mono text-[11px] text-muted-foreground">
                      {sample.id}
                    </span>
                    <span
                      className={cn(
                        "shrink-0 rounded-full px-1.5 py-0.5 text-[10px] font-medium",
                        sample.decision === "Blocked"
                          ? "bg-destructive/10 text-destructive"
                          : "bg-primary/10 text-primary",
                      )}
                    >
                      {sample.decision}
                    </span>
                  </span>
                  <span className="mt-1.5 block truncate text-sm font-medium">
                    {sample.model}
                  </span>
                  <span className="mt-0.5 block truncate text-xs text-muted-foreground">
                    {sample.project}
                  </span>
                </button>
              );
            })}
          </div>

          <div
            id="sample-panel"
            role="tabpanel"
            aria-labelledby={`sample-tab-${active.id}`}
            tabIndex={0}
            className="rounded-xl border bg-card p-5 focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none sm:p-6"
          >
            <div className="flex flex-wrap items-center gap-2 border-b pb-4">
              <h3 className="font-mono text-sm font-medium">{active.model}</h3>
              <span className="rounded-full border px-2 py-0.5 text-[11px] text-muted-foreground">
                {active.provider}
              </span>
            </div>

            <dl className="mt-4 divide-y">
              <DetailRow label="Project" value={active.project} />
              <DetailRow label="Team" value={active.team} />
              <DetailRow label="Policy" value={active.policy} />
              <DetailRow label="Timestamp" value={active.time} />
              <DetailRow label="Tokens" value={formatTokens(active.tokens)} />
              <DetailRow label="Cost avoided" value={formatCost(active.cost)} />
            </dl>

            <div className="mt-4 rounded-lg border border-dashed p-4">
              <p className="text-xs font-medium text-muted-foreground">Reason</p>
              <p className="mt-1.5 text-sm leading-relaxed text-pretty">{active.reason}</p>
            </div>
          </div>
        </div>

        <Note className="mt-6">{REQUEST_INSPECTOR.note}</Note>
      </Container>
    </section>
  );
}
