import test from "node:test";
import assert from "node:assert/strict";
import {
  formatUsd,
  formatUsdExact,
  microsToUsd,
  formatCompact,
  formatNumber,
  formatPercent,
  formatMs,
  formatDate,
} from "../lib/format.js";

test("Dashboard Widgets: currency and micros formatting", () => {
  assert.equal(microsToUsd(1_000_000), 1.0);
  assert.equal(microsToUsd(250_000), 0.25);
  assert.equal(formatUsd(1250), "$1.3k");
  assert.equal(formatUsd(2_500_000), "$2.5M");
  assert.equal(formatUsd(42), "$42");
  assert.equal(formatUsdExact(12.5), "$12.50");
});

test("Dashboard Widgets: token counts and compact numbers", () => {
  assert.equal(formatCompact(500), "500");
  assert.equal(formatCompact(1500), "1.5k");
  assert.equal(formatCompact(12_300_000), "12.3M");
  assert.equal(formatNumber(1234567), "1,234,567");
});

test("Dashboard Widgets: percentage and latency formatting", () => {
  assert.equal(formatPercent(0.125), "12.5%");
  assert.equal(formatPercent(0.999, 2), "99.90%");
  assert.equal(formatMs(125), "125ms");
  assert.equal(formatMs(1450), "1.45s");
});

test("Dashboard Widgets: date formatting enforces UTC", () => {
  const formatted = formatDate("2026-10-11T00:00:00Z");
  assert.ok(formatted.includes("Oct 11, 2026"));
});

interface OverviewState {
  totalRequests: number;
  totalTokens: number;
  totalCostMicros: number;
}

function calculateOverviewState(metrics: OverviewState) {
  const isEmpty = metrics.totalRequests === 0;
  return {
    isEmpty,
    requestsDisplay: formatNumber(metrics.totalRequests),
    tokensDisplay: formatCompact(metrics.totalTokens),
    costDisplay: formatUsd(microsToUsd(metrics.totalCostMicros)),
  };
}

test("Dashboard Widgets: empty vs populated overview states", () => {
  const empty = calculateOverviewState({ totalRequests: 0, totalTokens: 0, totalCostMicros: 0 });
  assert.equal(empty.isEmpty, true);
  assert.equal(empty.requestsDisplay, "0");
  assert.equal(empty.costDisplay, "$0");

  const populated = calculateOverviewState({ totalRequests: 45000, totalTokens: 12500000, totalCostMicros: 75000000 });
  assert.equal(populated.isEmpty, false);
  assert.equal(populated.requestsDisplay, "45,000");
  assert.equal(populated.tokensDisplay, "12.5M");
  assert.equal(populated.costDisplay, "$75");
});
