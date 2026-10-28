"use client";

import { useState } from "react";
import {
  BarChart3,
  CreditCard,
  PieChart,
  Globe,
  Sparkles,
  TrendingUp,
  Cpu,
  Layers,
} from "lucide-react";
import { AreaChart, BarChart, DonutChart, LineChart, chartColor } from "@/components/charts";
import { formatCompact, formatUsd } from "@/lib/format";
import type { TimeSeriesPoint, CategoryDatum } from "@/types";

type AnalyticsTab = "volume" | "spend" | "tokens" | "providers";

interface AnalyticsSuiteProps {
  requestSeries: readonly TimeSeriesPoint[];
  spendSeries: readonly TimeSeriesPoint[];
  tokenData: readonly CategoryDatum[];
  providerData: readonly CategoryDatum[];
  totalRequests: number;
  totalSpend: number;
  totalTokens: number;
  cachedTokens: number;
}

export function AnalyticsSuite({
  requestSeries,
  spendSeries,
  tokenData,
  providerData,
  totalRequests,
  totalSpend,
  totalTokens,
  cachedTokens,
}: AnalyticsSuiteProps) {
  const [activeTab, setActiveTab] = useState<AnalyticsTab>("volume");

  const cacheHitRatio = totalTokens > 0 ? (cachedTokens / totalTokens) * 100 : 0;

  return (
    <div className="rounded-2xl border border-border/70 bg-card/75 p-5 shadow-2xs backdrop-blur-xs">
      {/* Header with Visual Hierarchy & Tab Switcher (Hick's Law / Fitts's Law) */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b border-border/40 pb-4">
        <div>
          <div className="flex items-center gap-2">
            <h2 className="text-base font-semibold tracking-tight text-foreground">
              Traffic & Resource Intelligence
            </h2>
            <span className="rounded-full border border-border/60 bg-muted px-2.5 py-0.5 font-mono text-xs uppercase text-muted-foreground">
              Live Timeseries
            </span>
          </div>
          <p className="mt-1 text-sm text-muted-foreground">
            Correlate request velocity, estimated dollar costs, token ratios, and provider allocation.
          </p>
        </div>

        {/* Tab Controls (min 36px touch targets for Fitts's law) */}
        <div
          role="tablist"
          aria-label="Analytics view"
          className="inline-flex rounded-xl border border-border/60 bg-muted/40 p-1"
        >
          <button
            type="button"
            role="tab"
            aria-selected={activeTab === "volume"}
            onClick={() => setActiveTab("volume")}
            className={`inline-flex items-center gap-2 rounded-lg px-3.5 py-1.5 text-sm font-medium transition-all ${
              activeTab === "volume"
                ? "bg-card text-foreground shadow-xs font-semibold"
                : "text-muted-foreground hover:text-foreground"
            }`}
          >
            <BarChart3 className="size-4" />
            <span>Volume</span>
          </button>

          <button
            type="button"
            role="tab"
            aria-selected={activeTab === "spend"}
            onClick={() => setActiveTab("spend")}
            className={`inline-flex items-center gap-2 rounded-lg px-3.5 py-1.5 text-sm font-medium transition-all ${
              activeTab === "spend"
                ? "bg-card text-foreground shadow-xs font-semibold"
                : "text-muted-foreground hover:text-foreground"
            }`}
          >
            <CreditCard className="size-4" />
            <span>Spend</span>
          </button>

          <button
            type="button"
            role="tab"
            aria-selected={activeTab === "tokens"}
            onClick={() => setActiveTab("tokens")}
            className={`inline-flex items-center gap-2 rounded-lg px-3.5 py-1.5 text-sm font-medium transition-all ${
              activeTab === "tokens"
                ? "bg-card text-foreground shadow-xs font-semibold"
                : "text-muted-foreground hover:text-foreground"
            }`}
          >
            <PieChart className="size-4" />
            <span>Tokens</span>
          </button>

          <button
            type="button"
            role="tab"
            aria-selected={activeTab === "providers"}
            onClick={() => setActiveTab("providers")}
            className={`inline-flex items-center gap-2 rounded-lg px-3.5 py-1.5 text-sm font-medium transition-all ${
              activeTab === "providers"
                ? "bg-card text-foreground shadow-xs font-semibold"
                : "text-muted-foreground hover:text-foreground"
            }`}
          >
            <Globe className="size-4" />
            <span>Providers</span>
          </button>
        </div>
      </div>

      {/* Tab Panels with Instant Switch (Doherty Threshold < 100ms) */}
      <div className="mt-5">
        {/* Tab 1: Request Volume */}
        {activeTab === "volume" && (
          <div className="space-y-4">
            <div className="flex flex-wrap items-center justify-between gap-3 text-sm">
              <div className="flex items-center gap-2">
                <span className="text-muted-foreground">Total Inbound Requests:</span>
                <span className="font-semibold tabular-nums text-foreground">
                  {totalRequests.toLocaleString("en-US")} calls
                </span>
              </div>
              <span className="text-sm text-muted-foreground font-mono">
                Line: Request frequency per bucket
              </span>
            </div>

            {requestSeries.length > 0 ? (
              <LineChart data={requestSeries} color={chartColor(0)} formatValue={formatCompact} />
            ) : (
              <p className="py-12 text-center text-sm text-muted-foreground">
                No request volume recorded in this time range.
              </p>
            )}
          </div>
        )}

        {/* Tab 2: Estimated Spend */}
        {activeTab === "spend" && (
          <div className="space-y-4">
            <div className="flex flex-wrap items-center justify-between gap-3 text-sm">
              <div className="flex items-center gap-2">
                <span className="text-muted-foreground">Derived Total Spend:</span>
                <span className="font-semibold tabular-nums text-foreground">
                  {formatUsd(totalSpend)}
                </span>
              </div>
              <span className="text-sm text-muted-foreground font-mono">
                Area: Dollar cost computed from model token pricing
              </span>
            </div>

            {spendSeries.length > 0 ? (
              <AreaChart
                series={[{ label: "Spend", color: chartColor(1), data: spendSeries }]}
                formatValue={formatUsd}
              />
            ) : (
              <p className="py-12 text-center text-sm text-muted-foreground">
                No spend recorded in this time range.
              </p>
            )}
          </div>
        )}

        {/* Tab 3: Token Breakdown */}
        {activeTab === "tokens" && (
          <div className="space-y-5">
            <div className="flex flex-wrap items-center justify-between gap-3 text-sm">
              <div className="flex items-center gap-2">
                <span className="text-muted-foreground">Total Tokens Processed:</span>
                <span className="font-semibold tabular-nums text-foreground">
                  {formatCompact(totalTokens)} tokens
                </span>
              </div>
              <div className="flex items-center gap-2">
                <span className="inline-flex items-center gap-1.5 rounded-full bg-emerald-500/10 px-3 py-1 text-xs font-semibold text-emerald-500 border border-emerald-500/20">
                  <Sparkles className="size-3.5" />
                  Cache Efficiency: {cacheHitRatio.toFixed(1)}%
                </span>
              </div>
            </div>

            {tokenData.length > 0 ? (
              <div className="flex flex-col sm:flex-row items-center justify-around gap-6 pt-2 pb-4">
                <DonutChart data={tokenData} centerLabel="tokens" />
                <div className="grid grid-cols-3 gap-5 w-full sm:w-auto text-center sm:text-left">
                  {tokenData.map((d, i) => (
                    <div key={d.label} className="border-l-2 border-primary/40 pl-3.5">
                      <p className="text-xs font-semibold uppercase tracking-wider text-muted-foreground font-mono">
                        {d.label}
                      </p>
                      <p className="text-xl font-bold tabular-nums text-foreground mt-1">
                        {formatCompact(d.value)}
                      </p>
                    </div>
                  ))}
                </div>
              </div>
            ) : (
              <p className="py-12 text-center text-sm text-muted-foreground">
                No token composition telemetry recorded yet.
              </p>
            )}
          </div>
        )}

        {/* Tab 4: Provider Distribution */}
        {activeTab === "providers" && (
          <div className="space-y-4">
            <div className="flex flex-wrap items-center justify-between gap-3 text-sm">
              <span className="text-muted-foreground font-medium">Upstream Platform Distribution</span>
              <span className="text-sm text-muted-foreground font-mono">
                Sorted by request volume
              </span>
            </div>

            {providerData.length > 0 ? (
              <BarChart data={providerData} formatValue={formatCompact} />
            ) : (
              <p className="py-12 text-center text-sm text-muted-foreground">
                No provider routing recorded yet.
              </p>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
