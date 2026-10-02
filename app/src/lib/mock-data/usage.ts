import { models } from "./models";
import { NOW } from "./clock";
import { projects } from "./projects";
import { providers } from "./providers";
import { teams } from "./teams";
import { users } from "./users";
import type {
  TimeSeriesPoint,
  UsageBreakdown,
  UsageSeries,
  UsageTotals,
} from "@/types";

function mulberry32(seed: number) {
  let a = seed;
  return () => {
    a |= 0;
    a = (a + 0x6d2b79f5) | 0;
    let t = Math.imul(a ^ (a >>> 15), 1 | a);
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

const DAY = 86_400_000;
const isoDay = (offset: number) =>
  new Date(NOW.getTime() - offset * DAY).toISOString().slice(0, 10);

const isoHour = (offset: number) => {
  const d = new Date(NOW.getTime() - offset * 3_600_000);
  return `${String(d.getUTCHours()).padStart(2, "0")}:00`;
};

const sum = (points: TimeSeriesPoint[]) =>
  points.reduce((acc, point) => acc + point.value, 0);

/** Traffic is lower at weekends and outside working hours. */
const weekendFactor = (iso: string) => {
  const day = new Date(`${iso}T00:00:00Z`).getUTCDay();
  return day === 0 || day === 6 ? 0.58 : 1;
};

const buildDaily = (seed: number) => {
  const rand = mulberry32(seed);
  const calls: TimeSeriesPoint[] = [];
  const inputTokens: TimeSeriesPoint[] = [];
  const outputTokens: TimeSeriesPoint[] = [];
  const cost: TimeSeriesPoint[] = [];

  for (let offset = 29; offset >= 0; offset -= 1) {
    const date = isoDay(offset);
    const factor = weekendFactor(date);
    // Gentle upward trend plus bounded noise.
    const trend = 1 + (29 - offset) * 0.006;
    const callCount = Math.round((48_000 + rand() * 14_000) * factor * trend);
    const input = Math.round(callCount * (760 + rand() * 320));
    const output = Math.round(callCount * (210 + rand() * 130));
    calls.push({ date, value: callCount });
    inputTokens.push({ date, value: input });
    outputTokens.push({ date, value: output });
    cost.push({
      date,
      value:
        Math.round(
          (input / 1_000_000) * 1.9 + (output / 1_000_000) * 7.4 + 40,
        ),
    });
  }
  return { calls, inputTokens, outputTokens, cost };
};

const buildHourly = (seed: number) => {
  const rand = mulberry32(seed);
  const calls: TimeSeriesPoint[] = [];
  const inputTokens: TimeSeriesPoint[] = [];
  const outputTokens: TimeSeriesPoint[] = [];
  const cost: TimeSeriesPoint[] = [];

  for (let offset = 23; offset >= 0; offset -= 1) {
    const label = isoHour(offset);
    const hour = Number(label.slice(0, 2));
    // Working-hours hump peaking around 14:00 UTC.
    const shape = 0.35 + Math.cos(((hour - 14) / 24) * 2 * Math.PI) * 0.65;
    const callCount = Math.round((2_000 + rand() * 700) * Math.max(0.2, shape));
    const input = Math.round(callCount * (800 + rand() * 260));
    const output = Math.round(callCount * (220 + rand() * 110));
    calls.push({ date: label, value: callCount });
    inputTokens.push({ date: label, value: input });
    outputTokens.push({ date: label, value: output });
    cost.push({
      date: label,
      value:
        Math.round(
          (input / 1_000_000) * 1.9 + (output / 1_000_000) * 7.4 + 1.5,
        ),
    });
  }
  return { calls, inputTokens, outputTokens, cost };
};

/** Totals are derived from the series, so the two can never disagree. */
const totalsFrom = (parts: {
  calls: TimeSeriesPoint[];
  inputTokens: TimeSeriesPoint[];
  outputTokens: TimeSeriesPoint[];
  cost: TimeSeriesPoint[];
}): UsageTotals => {
  const input = sum(parts.inputTokens);
  const output = sum(parts.outputTokens);
  return {
    totalCalls: sum(parts.calls),
    inputTokens: input,
    outputTokens: output,
    totalTokens: input + output,
    cost: sum(parts.cost),
    p50LatencyMs: 486,
    p95LatencyMs: 1_642,
    errorRate: 0.021,
  };
};

const daily = buildDaily(20260929);
const hourly = buildHourly(777);

export const usageSeries: UsageSeries = {
  granularity: "day",
  totals: totalsFrom(daily),
  calls: daily.calls,
  tokens: daily.calls.map((point, index) => ({
    date: point.date,
    value: daily.inputTokens[index]!.value + daily.outputTokens[index]!.value,
  })),
  cost: daily.cost,
  inputTokens: daily.inputTokens,
  outputTokens: daily.outputTokens,
};

/** Rolling 24h window for the live activity view. */
export const liveUsageSeries: UsageSeries = {
  granularity: "hour",
  totals: totalsFrom(hourly),
  calls: hourly.calls,
  tokens: hourly.calls.map((point, index) => ({
    date: point.date,
    value: hourly.inputTokens[index]!.value + hourly.outputTokens[index]!.value,
  })),
  cost: hourly.cost,
  inputTokens: hourly.inputTokens,
  outputTokens: hourly.outputTokens,
};

export const usageBreakdown: UsageBreakdown = {
  byModel: models
    .filter((model) => model.callCount30d > 0)
    .map((model) => ({
      label: model.displayName,
      value: model.spend30d,
      modelId: model.id,
      cost: model.spend30d,
      calls: model.callCount30d,
    }))
    .sort((a, b) => b.value - a.value),
  byProvider: providers
    .map((provider) => {
      const owned = models.filter((model) => model.providerId === provider.id);
      return {
        label: provider.displayName,
        value: owned.reduce((acc, model) => acc + model.spend30d, 0),
        providerId: provider.id,
        cost: owned.reduce((acc, model) => acc + model.spend30d, 0),
        calls: owned.reduce((acc, model) => acc + model.callCount30d, 0),
      };
    })
    .sort((a, b) => b.value - a.value),
  byProject: [...projects]
    .sort((a, b) => b.spendMtd - a.spendMtd)
    .map((project) => ({ label: project.name, value: project.spendMtd })),
  byTeam: [...teams]
    .sort((a, b) => b.spendMtd - a.spendMtd)
    .map((team) => ({ label: team.name, value: team.spendMtd })),
  byUser: users
    .filter((user) => user.spendMtd > 0)
    .sort((a, b) => b.spendMtd - a.spendMtd)
    .slice(0, 12)
    .map((user) => ({ label: user.name, value: user.spendMtd })),
};
