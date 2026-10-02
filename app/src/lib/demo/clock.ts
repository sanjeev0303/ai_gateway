/**
 * Single clock for the whole demo dataset.
 *
 * Every relative timestamp is derived from this constant rather than the wall
 * clock, so the product renders identically on the server and the client and
 * never shifts between a server render and a hydration pass. It lives in its own
 * leaf module so `entities.ts` and `traffic.ts` can both read it without an
 * import cycle.
 */

/** 30 Sep 2026, 14:20 UTC. "Today" is therefore a partial 14-hour day. */
export const DEMO_NOW = new Date("2026-09-30T14:20:00Z");

export const DEMO_NOW_ISO = DEMO_NOW.toISOString();

const MINUTE = 60_000;
const HOUR = 3_600_000;
const DAY = 86_400_000;

export const demoMinutesAgo = (minutes: number): string =>
  new Date(DEMO_NOW.getTime() - minutes * MINUTE).toISOString();

export const demoHoursAgo = (hours: number): string =>
  new Date(DEMO_NOW.getTime() - hours * HOUR).toISOString();

export const demoDaysAgo = (days: number): string =>
  new Date(DEMO_NOW.getTime() - days * DAY).toISOString();

export const demoDaysFromNow = (days: number): string =>
  new Date(DEMO_NOW.getTime() + days * DAY).toISOString();

/** "YYYY-MM-DD" for a day `offset` days before today, offset 0 being today. */
export const demoDayKey = (offset: number): string =>
  new Date(DEMO_NOW.getTime() - offset * DAY).toISOString().slice(0, 10);

/** Short label like "Sep 28" for headers and axis ticks. */
export function demoDayLabel(dayKey: string): string {
  return new Date(`${dayKey}T00:00:00Z`).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    timeZone: "UTC",
  });
}
