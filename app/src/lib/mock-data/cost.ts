import { models } from "./models";
import { NOW } from "./clock";
import { projects } from "./projects";
import { providers } from "./providers";
import { teams } from "./teams";
import { usageSeries } from "./usage";
import type { CostSeries, CostSummary, TimeSeriesPoint } from "@/types";

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
const WINDOW_DAYS = 60;

const isoDay = (offset: number) =>
  new Date(NOW.getTime() - offset * DAY).toISOString().slice(0, 10);

const weekendFactor = (iso: string) => {
  const day = new Date(`${iso}T00:00:00Z`).getUTCDay();
  return day === 0 || day === 6 ? 0.58 : 1;
};

/** 60 daily points so month-over-month comparison has a full prior month. */
const dailySpend: TimeSeriesPoint[] = (() => {
  const rand = mulberry32(4242);
  const points: TimeSeriesPoint[] = [];
  for (let offset = WINDOW_DAYS - 1; offset >= 0; offset -= 1) {
    const date = isoDay(offset);
    const trend = 1 + (WINDOW_DAYS - offset) * 0.0042;
    const jitter = 0.92 + rand() * 0.16;
    points.push({
      date,
      value: Math.round((3_180 + rand() * 780) * weekendFactor(date) * trend * jitter),
    });
  }
  return points;
})();

const inMonth = (points: TimeSeriesPoint[], month: string) =>
  points.filter((point) => point.date.startsWith(month));

const total = (points: TimeSeriesPoint[]) =>
  points.reduce((acc, point) => acc + point.value, 0);

/** Month-to-date covers 1-29 Sep, the prior comparison covers all of August. */
const currentMonth = inMonth(dailySpend, "2026-09");
const priorMonth = inMonth(dailySpend, "2026-08");

const mtd = total(currentMonth);
const lastMonth = total(priorMonth);

// Compare daily averages so a 29-day month is not penalised against a 31-day one.
const currentDaily = mtd / currentMonth.length;
const priorDaily = lastMonth / priorMonth.length;
const changePct = (currentDaily - priorDaily) / priorDaily;

const MONTHLY_BUDGET = 20_000;
const projected = Math.round(mtd * 1.04);

export const costSummary: CostSummary = {
  mtd,
  lastMonth,
  changePct,
  forecast: {
    projected,
    actual: mtd,
    budget: MONTHLY_BUDGET,
    projectedOverspend: Math.max(0, projected - MONTHLY_BUDGET),
  },
  blendedCostPerMTok: Number(
    (mtd / (usageSeries.totals.totalTokens / 1_000_000)).toFixed(2),
  ),
};

export const costSeries: CostSeries = {
  dailySpend,
  byModel: [...models]
    .filter((model) => model.spend30d > 0)
    .sort((a, b) => b.spend30d - a.spend30d)
    .map((model) => ({ label: model.displayName, value: model.spend30d })),
  byProvider: providers
    .map((provider) => ({
      label: provider.displayName,
      value: models
        .filter((model) => model.providerId === provider.id)
        .reduce((acc, model) => acc + model.spend30d, 0),
    }))
    .sort((a, b) => b.value - a.value),
  byTeam: [...teams]
    .sort((a, b) => b.spendMtd - a.spendMtd)
    .map((team) => ({ label: team.name, value: team.spendMtd })),
  byProject: [...projects]
    .sort((a, b) => b.spendMtd - a.spendMtd)
    .map((project) => ({ label: project.name, value: project.spendMtd })),
  byEnvironment: ["production", "staging", "development"].map(
    (environment) => ({
      label: environment,
      value: projects
        .filter((project) => project.environment === environment)
        .reduce((acc, project) => acc + project.spendMtd, 0),
    }),
  ),
};
