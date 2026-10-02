"use client";

import { useEffect, useRef, useState } from "react";
import { Check, Copy } from "lucide-react";

import { Container, Note, SectionHeading } from "@/components/landing/section";
import { Button } from "@/components/ui/button";
import { CODE_TABS } from "@/lib/landing-data";
import { cn } from "@/lib/utils";

export function CodeTabs() {
  const samples = CODE_TABS.samples;
  const [activeId, setActiveId] = useState(samples[0]?.id ?? "");
  const [copied, setCopied] = useState(false);
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const tabRefs = useRef<Record<string, HTMLButtonElement | null>>({});

  const active = samples.find((sample) => sample.id === activeId) ?? samples[0];

  useEffect(() => {
    return () => {
      if (timer.current) clearTimeout(timer.current);
    };
  }, []);

  if (!active) return null;

  async function copy() {
    if (!active) return;
    try {
      await navigator.clipboard.writeText(active.code);
      setCopied(true);
      if (timer.current) clearTimeout(timer.current);
      timer.current = setTimeout(() => setCopied(false), 2000);
    } catch {
      setCopied(false);
    }
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
    <section className="border-b">
      <Container className="py-20 sm:py-24 lg:py-28">
        <SectionHeading
          eyebrow={CODE_TABS.eyebrow}
          title={CODE_TABS.headline}
          body={CODE_TABS.body}
        />

        <div className="mt-12 overflow-hidden rounded-xl border bg-card">
          <div className="flex flex-wrap items-center justify-between gap-2 border-b bg-muted/30 px-3 py-2">
            <div role="tablist" aria-label="Language" className="flex flex-wrap gap-1">
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
                    ref={(node) => {
                      tabRefs.current[sample.id] = node;
                    }}
                    onClick={() => setActiveId(sample.id)}
                    onKeyDown={(event) => onKeyDown(event, index)}
                    className={cn(
                      "rounded-md px-3 py-1.5 text-xs font-medium transition-colors",
                      "focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none",
                      selected
                        ? "bg-background text-foreground shadow-sm"
                        : "text-muted-foreground hover:text-foreground",
                    )}
                  >
                    {sample.label}
                  </button>
                );
              })}
            </div>

            <Button type="button" variant="outline" size="sm" onClick={copy} className="gap-1.5">
              {copied ? (
                <Check className="size-3.5" aria-hidden />
              ) : (
                <Copy className="size-3.5" aria-hidden />
              )}
              {copied ? "Copied" : "Copy"}
            </Button>
          </div>

          <div
            id="code-panel"
            role="tabpanel"
            aria-label={`${active.label} example`}
            className="overflow-x-auto"
          >
            <pre className="p-4 font-mono text-xs leading-relaxed sm:p-5 sm:text-[13px]">
              <code>{active.code}</code>
            </pre>
          </div>

          <p aria-live="polite" className="sr-only">
            {copied ? `${active.label} snippet copied to clipboard` : ""}
          </p>
        </div>

        <Note className="mt-6">{CODE_TABS.note}</Note>
      </Container>
    </section>
  );
}
