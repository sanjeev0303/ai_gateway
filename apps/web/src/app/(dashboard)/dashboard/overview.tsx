"use client";

import { useState } from "react";

import { ChartSlot, RankingSlot, TimeRange, defaultRange, type TimeRangeValue } from "@/lib/overview-parts";
import { formatCount, formatMs, formatPercent, formatUsd, KpiCard } from "@/lib/metrics";
import { ProviderHealthPanel, SetupPanel } from "@/lib/platform-panels";

const NEEDS_AGGREGATES = ["GET /v1/usage/aggregates", "GET /v1/usage/daily"] as const;
const NEEDS_ACTIVITY = ["GET /v1/requests (llm_requests feed)"] as const;

export function Overview() {
  const [range, setRange] = useState<TimeRangeValue>(defaultRange);

  return (
    <>
      <header className="dash-head">
        <h1>Overview</h1>
        <p className="dash-note">
          Organization usage, spend, and reliability. Figures appear once the gateway records
          traffic.
        </p>
        <TimeRange value={range} onChange={setRange} />
      </header>

      <section className="dash-panel">
        <h2>Headline metrics</h2>
        <p className="dash-note">
          Every metric below needs a server-side aggregate. Until the control API exposes one, the
          dashboard reports unknown rather than zero.
        </p>
        <div className="dash-tiles">
          <KpiCard
            label="Requests"
            current={null}
            previous={null}
            format={formatCount}
            lowerIsBetter={false}
          />
          <KpiCard
            label="Tokens"
            current={null}
            previous={null}
            format={formatCount}
            lowerIsBetter={false}
          />
          <KpiCard
            label="AI spend"
            current={null}
            previous={null}
            format={formatUsd}
            lowerIsBetter
          />
          <KpiCard
            label="Active users"
            current={null}
            previous={null}
            format={formatCount}
            lowerIsBetter={false}
          />
          <KpiCard
            label="Error rate"
            current={null}
            previous={null}
            format={formatPercent}
            lowerIsBetter
          />
          <KpiCard label="P95 TTFT" current={null} previous={null} format={formatMs} lowerIsBetter />
        </div>
      </section>

      <section className="dash-panel">
        <h2>Charts</h2>
        <div className="chart-grid">
          <ChartSlot title="Request volume" needs={NEEDS_AGGREGATES} />
          <ChartSlot title="Token usage" needs={NEEDS_AGGREGATES} />
          <ChartSlot title="AI spend" needs={NEEDS_AGGREGATES} />
          <ChartSlot title="Provider distribution" needs={NEEDS_AGGREGATES} />
          <ChartSlot title="Model distribution" needs={NEEDS_AGGREGATES} />
        </div>
      </section>

      <section className="dash-panel">
        <h2>Live activity</h2>
        <p className="dash-note">
          Recent gateway calls. Requires a request log endpoint and a realtime stream; neither is
          exposed by the control API today.
        </p>
        <div className="chart-grid">
          <RankingSlot
            title="Most active users"
            basis="Ranked by request count."
            needs={NEEDS_ACTIVITY}
          />
          <RankingSlot title="Top projects" basis="Ranked by spend." needs={NEEDS_ACTIVITY} />
        </div>
      </section>

      <section className="dash-panel">
        <h2>Rankings</h2>
        <div className="chart-grid">
          <RankingSlot title="Most used models" basis="Ranked by tokens." needs={NEEDS_AGGREGATES} />
          <RankingSlot
            title="Most used providers"
            basis="Ranked by requests."
            needs={NEEDS_AGGREGATES}
          />
        </div>
      </section>

      <ProviderHealthPanel />
      <SetupPanel />
    </>
  );
}
