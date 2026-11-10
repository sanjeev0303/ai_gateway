"use client";

import { useState } from "react";
import Link from "next/link";
import { ArrowRight } from "lucide-react";
import { formatCompact, formatUsd } from "@ai-governance/console-ui/lib/format";
import { cn } from "@ai-governance/console-ui/lib/utils";

export interface RankRow {
  name: string;
  requests: number;
  /** Models carry spend; providers do not. */
  spend?: number;
}

type View = "models" | "providers";

/**
 * Where requests go: the top models or providers as ranked bars, each with its
 * share of all requests. One question, one bar length, the number beside it.
 */
export function RankingCard({ models, providers }: { models: readonly RankRow[]; providers: readonly RankRow[] }) {
  const [view, setView] = useState<View>("models");
  const rows = view === "models" ? models : providers;
  const total = rows.reduce((sum, r) => sum + r.requests, 0);
  const href = view === "models" ? "/app/observe/models" : "/app/observe/providers";

  return (
    <section aria-labelledby="ranking-heading" className="flex flex-col gap-4 rounded-2xl border border-border bg-card p-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 id="ranking-heading" className="text-[0.9375rem] font-semibold tracking-[-0.01em]">
            Where requests go
          </h2>
          <p className="text-xs text-muted-foreground">Share of requests in this range</p>
        </div>
        <div role="tablist" aria-label="Group by" className="liquid-glass inline-flex h-8 items-center rounded-full border p-[3px]">
          {(["models", "providers"] as const).map((key) => (
            <button
              key={key}
              type="button"
              role="tab"
              aria-selected={view === key}
              onClick={() => setView(key)}
              className={cn(
                "h-full rounded-full px-3 text-xs font-medium capitalize transition-[background-color,color] duration-150",
                view === key ? "bg-primary text-primary-foreground" : "text-muted-foreground hover:text-foreground",
              )}
            >
              {key}
            </button>
          ))}
        </div>
      </div>

      {rows.length === 0 ? (
        <div className="flex flex-1 flex-col items-center justify-center gap-1 py-10 text-center">
          <p className="text-sm font-semibold">No {view} used in this range</p>
          <p className="text-sm text-muted-foreground">They are ranked here once requests go through the gateway.</p>
        </div>
      ) : (
        <ol className="flex flex-col gap-3.5">
          {rows.map((row, i) => {
            const share = total > 0 ? row.requests / total : 0;
            return (
              <li key={row.name} className="flex flex-col gap-1.5">
                <div className="flex items-baseline justify-between gap-3 text-sm">
                  <span className="flex min-w-0 items-baseline gap-2">
                    <span className="w-4 shrink-0 text-xs text-muted-foreground tabular-nums">{i + 1}</span>
                    <span className="truncate font-medium">{row.name}</span>
                  </span>
                  <span className="shrink-0 text-right tabular-nums">
                    <span className="font-semibold">{Math.round(share * 100)}%</span>
                    <span className="ml-2 text-xs text-muted-foreground">
                      {formatCompact(row.requests)} req{row.spend != null ? ` · ${formatUsd(row.spend)}` : ""}
                    </span>
                  </span>
                </div>
                <div className="ml-6 h-1.5 overflow-hidden rounded-full bg-muted" aria-hidden>
                  <div className={cn("h-full rounded-full", i === 0 ? "bg-primary" : "bg-primary/45")} style={{ width: `${Math.max(2, share * 100)}%` }} />
                </div>
              </li>
            );
          })}
        </ol>
      )}

      <Link
        href={href}
        className="mt-auto inline-flex w-fit items-center gap-1 text-sm font-medium text-primary underline-offset-4 hover:underline focus-visible:ring-3 focus-visible:ring-ring/40 focus-visible:outline-none"
      >
        All {view}
        <ArrowRight className="size-3.5" aria-hidden />
      </Link>
    </section>
  );
}
