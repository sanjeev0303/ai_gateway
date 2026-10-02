"use client";

import { useState } from "react";

import { Container, Note, SectionHeading } from "@/components/landing/section";
import { ATTRIBUTION, type AttributionMetric } from "@/lib/landing-data";
import { cn } from "@/lib/utils";

export function Attribution() {
  const metrics = ATTRIBUTION.metrics;
  const [activeId, setActiveId] = useState<AttributionMetric>("spend");
  const active = metrics.find((metric) => metric.id === activeId) ?? metrics[0];

  if (!active) return null;

  const values = ATTRIBUTION.rows.map((row) => row[active.id]);
  const max = Math.max(...values, 1);

  return (
    <section className="border-b">
      <Container className="py-20 sm:py-24 lg:py-28">
        <SectionHeading
          eyebrow={ATTRIBUTION.eyebrow}
          title={ATTRIBUTION.headline}
          body={ATTRIBUTION.body}
        />

        <div
          role="group"
          aria-label="Attribution metric"
          className="mt-10 inline-flex rounded-lg border bg-card p-1"
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
                  "rounded-md px-4 py-1.5 text-sm font-medium transition-colors",
                  "focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none",
                  selected
                    ? "bg-primary text-primary-foreground"
                    : "text-muted-foreground hover:text-foreground",
                )}
              >
                {metric.label}
              </button>
            );
          })}
        </div>

        <p className="mt-4 text-sm text-muted-foreground">{active.caption}</p>

        <div className="mt-8 rounded-xl border bg-card p-5 sm:p-6">
          <div className="flex items-baseline justify-between border-b pb-3">
            <span className="text-xs font-medium text-muted-foreground">Team</span>
            <span className="text-xs font-medium text-muted-foreground">
              {active.label} ({active.unit})
            </span>
          </div>

          <ul className="mt-4 space-y-3.5">
            {ATTRIBUTION.rows.map((row) => {
              const value = row[active.id];
              const width = Math.max((value / max) * 100, 2);
              return (
                <li key={row.label}>
                  <div className="flex items-baseline justify-between gap-4">
                    <span className="truncate text-sm">{row.label}</span>
                    <span className="shrink-0 text-sm font-medium tabular-nums">
                      {active.format(value)}
                    </span>
                  </div>
                  {/* The bar is decorative; the sr-only table below carries the data. */}
                  <div
                    aria-hidden="true"
                    className="mt-1.5 h-1.5 w-full overflow-hidden rounded-full bg-muted"
                  >
                    <div
                      className="h-full rounded-full bg-primary"
                      style={{ width: `${width}%` }}
                    />
                  </div>
                </li>
              );
            })}
          </ul>
        </div>

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

        <Note className="mt-6">{ATTRIBUTION.note}</Note>
      </Container>
    </section>
  );
}
