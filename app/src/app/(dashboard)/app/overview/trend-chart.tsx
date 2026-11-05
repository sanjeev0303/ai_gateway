"use client";

import { useEffect, useId, useMemo, useRef, useState } from "react";
import { formatCompact, formatUsd } from "@/lib/format";
import { cn } from "@/lib/utils";

export interface TrendPoint {
  /** ISO start of the bucket. */
  at: string;
  requests: number;
  spend: number;
}

type Metric = "requests" | "spend";

const METRICS: Record<Metric, { label: string; color: string; format: (n: number) => string; noun: string }> = {
  requests: { label: "Requests", color: "var(--primary)", format: formatCompact, noun: "requests" },
  spend: { label: "Spend", color: "var(--chart-1)", format: formatUsd, noun: "spent" },
};

const H = 220;
const PAD = { top: 12, right: 8, bottom: 26, left: 44 };

/** Monotone cubic path through the points (Fritsch-Carlson), so the curve never dips below zero. */
function smoothPath(points: { x: number; y: number }[]): string {
  const n = points.length;
  if (n === 0) return "";
  if (n === 1) return `M${points[0]!.x} ${points[0]!.y}`;
  const dx: number[] = [];
  const slope: number[] = [];
  for (let i = 0; i < n - 1; i++) {
    dx.push(points[i + 1]!.x - points[i]!.x);
    slope.push((points[i + 1]!.y - points[i]!.y) / dx[i]!);
  }
  const tangent: number[] = [slope[0]!];
  for (let i = 1; i < n - 1; i++) {
    const a = slope[i - 1]!;
    const b = slope[i]!;
    tangent.push(a * b <= 0 ? 0 : (3 * (dx[i - 1]! + dx[i]!)) / ((2 * dx[i]! + dx[i - 1]!) / a + (dx[i]! + 2 * dx[i - 1]!) / b));
  }
  tangent.push(slope[n - 2]!);
  let d = `M${points[0]!.x} ${points[0]!.y}`;
  for (let i = 0; i < n - 1; i++) {
    const p = points[i]!;
    const q = points[i + 1]!;
    const h = dx[i]! / 3;
    d += `C${p.x + h} ${p.y + h * tangent[i]!} ${q.x - h} ${q.y - h * tangent[i + 1]!} ${q.x} ${q.y}`;
  }
  return d;
}

/** A round top for the y axis, close above the peak so the line uses the height. */
function niceTop(max: number): number {
  if (max <= 0) return 1;
  const power = 10 ** Math.floor(Math.log10(max));
  for (const step of [1, 1.5, 2, 2.5, 3, 4, 5, 6, 8, 10]) if (step * power >= max) return step * power;
  return 10 * power;
}

/**
 * Requests or spend over the selected range, as one smooth area. The headline
 * says what the chart shows (the total and the busiest moment); hovering, or the
 * arrow keys, read any point exactly.
 */
export function TrendChart({ points }: { points: readonly TrendPoint[] }) {
  const [metric, setMetric] = useState<Metric>("requests");
  const [active, setActive] = useState<number | null>(null);
  const gradientId = useId();
  const m = METRICS[metric];

  // The API fills empty buckets with zeros, so "no traffic" is a zero total, not an empty list.
  const hasData = points.some((p) => p.requests > 0 || p.spend > 0);

  // The viewBox follows the real width, so one unit is one pixel and the labels stay 12px at any size.
  const frameRef = useRef<HTMLDivElement>(null);
  const [W, setW] = useState(720);
  useEffect(() => {
    const frame = frameRef.current;
    if (!frame) return;
    const observer = new ResizeObserver(([entry]) => {
      if (entry) setW(Math.max(280, Math.round(entry.contentRect.width)));
    });
    observer.observe(frame);
    return () => observer.disconnect();
  }, [hasData]);

  const spanHours = points.length > 1 ? (Date.parse(points.at(-1)!.at) - Date.parse(points[0]!.at)) / 3_600_000 : 0;
  const tick = useMemo(
    () =>
      new Intl.DateTimeFormat("en-US", spanHours <= 48 ? { hour: "numeric", minute: "2-digit" } : { month: "short", day: "numeric" }),
    [spanHours],
  );
  // Daily buckets have no meaningful time of day, so they are named by date alone.
  const daily = points.length > 1 && Date.parse(points[1]!.at) - Date.parse(points[0]!.at) >= 86_400_000;
  const full = useMemo(
    () =>
      new Intl.DateTimeFormat(
        "en-US",
        daily ? { weekday: "short", month: "short", day: "numeric" } : { month: "short", day: "numeric", hour: "numeric", minute: "2-digit" },
      ),
    [daily],
  );

  if (!hasData) {
    return (
      <div className="flex h-[260px] flex-col items-center justify-center gap-1 text-center">
        <p className="text-sm font-semibold">No traffic in this range</p>
        <p className="text-sm text-muted-foreground">Requests and spend appear here once apps call the gateway.</p>
      </div>
    );
  }

  const values = points.map((p) => p[metric]);
  const total = values.reduce((a, b) => a + b, 0);
  const peakIndex = values.indexOf(Math.max(...values));
  const top = niceTop(Math.max(...values));
  const plotW = W - PAD.left - PAD.right;
  const plotH = H - PAD.top - PAD.bottom;
  const x = (i: number) => PAD.left + (points.length === 1 ? plotW / 2 : (i / (points.length - 1)) * plotW);
  const y = (v: number) => PAD.top + plotH - (v / top) * plotH;
  const xy = values.map((v, i) => ({ x: x(i), y: y(v) }));
  const line = smoothPath(xy);
  const area = `${line}L${xy.at(-1)!.x} ${PAD.top + plotH}L${xy[0]!.x} ${PAD.top + plotH}Z`;
  // A half-way line only when its label is exact: "2.5 requests" would print as a rounded 3.
  const gridValues = metric === "spend" || top % 2 === 0 ? [0, top / 2, top] : [0, top];
  const labelEvery = Math.max(1, Math.ceil(points.length / Math.max(2, Math.floor(W / 110))));
  const shown = active ?? null;

  const pick = (clientX: number, rect: DOMRect) => {
    const ratio = (clientX - rect.left) / rect.width;
    const svgX = ratio * W;
    const i = Math.round(((svgX - PAD.left) / plotW) * (points.length - 1));
    setActive(Math.min(points.length - 1, Math.max(0, i)));
  };

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <p className="font-display text-[1.75rem] leading-none tabular-nums">{m.format(total)}</p>
          <p className="mt-1.5 text-sm text-muted-foreground">
            {m.label === "Spend" ? "Spent" : "Requests"} in this range · busiest {full.format(new Date(points[peakIndex]!.at))} ({m.format(values[peakIndex]!)})
          </p>
        </div>
        <div role="tablist" aria-label="Metric" className="liquid-glass inline-flex h-9 items-center rounded-full border p-[3px]">
          {(Object.keys(METRICS) as Metric[]).map((key) => (
            <button
              key={key}
              type="button"
              role="tab"
              aria-selected={metric === key}
              onClick={() => setMetric(key)}
              className={cn(
                "h-full rounded-full px-3.5 text-sm font-medium transition-[background-color,color] duration-150",
                metric === key ? "bg-primary text-primary-foreground" : "text-muted-foreground hover:text-foreground",
              )}
            >
              {METRICS[key].label}
            </button>
          ))}
        </div>
      </div>

      <div ref={frameRef} className="relative">
        <svg
          viewBox={`0 0 ${W} ${H}`}
          className="h-auto w-full touch-none outline-none focus-visible:ring-3 focus-visible:ring-ring/40 rounded-lg"
          role="img"
          tabIndex={0}
          aria-label={`${m.label} over the range: ${m.format(total)} in total, busiest at ${full.format(new Date(points[peakIndex]!.at))} with ${m.format(values[peakIndex]!)}. Use the arrow keys to read each point.`}
          onPointerMove={(e) => pick(e.clientX, e.currentTarget.getBoundingClientRect())}
          onPointerLeave={() => setActive(null)}
          onKeyDown={(e) => {
            if (e.key === "ArrowRight") setActive((i) => Math.min(points.length - 1, (i ?? -1) + 1));
            else if (e.key === "ArrowLeft") setActive((i) => Math.max(0, (i ?? points.length) - 1));
            else if (e.key === "Escape") setActive(null);
            else return;
            e.preventDefault();
          }}
          onBlur={() => setActive(null)}
        >
          <defs>
            <linearGradient id={gradientId} x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor={m.color} stopOpacity={0.22} />
              <stop offset="100%" stopColor={m.color} stopOpacity={0} />
            </linearGradient>
          </defs>
          {gridValues.map((v) => (
            <g key={v}>
              <line x1={PAD.left} x2={W - PAD.right} y1={y(v)} y2={y(v)} stroke="var(--border)" strokeDasharray={v === 0 ? undefined : "3 4"} />
              <text x={PAD.left - 8} y={y(v)} textAnchor="end" dominantBaseline="middle" className="fill-muted-foreground text-[12px] tabular-nums">
                {m.format(v)}
              </text>
            </g>
          ))}
          <path d={area} fill={`url(#${gradientId})`} />
          <path d={line} fill="none" stroke={m.color} strokeWidth={2.25} strokeLinecap="round" strokeLinejoin="round" />
          {points.map((p, i) =>
            i % labelEvery === 0 ? (
              <text key={p.at} x={x(i)} y={H - 6} textAnchor="middle" className="fill-muted-foreground text-[12px]">
                {tick.format(new Date(p.at))}
              </text>
            ) : null,
          )}
          {/* The busiest point is always marked; the hovered one gets a guide line */}
          <circle cx={xy[peakIndex]!.x} cy={xy[peakIndex]!.y} r={3.5} fill="var(--card)" stroke={m.color} strokeWidth={2} />
          {shown !== null && (
            <g>
              <line x1={xy[shown]!.x} x2={xy[shown]!.x} y1={PAD.top} y2={PAD.top + plotH} stroke="var(--border-strong)" />
              <circle cx={xy[shown]!.x} cy={xy[shown]!.y} r={4.5} fill={m.color} stroke="var(--card)" strokeWidth={2} />
            </g>
          )}
        </svg>
        {shown !== null && (
          <div
            className="liquid-glass-panel pointer-events-none absolute top-1 rounded-lg border px-2.5 py-1.5 text-xs"
            style={{ left: `${(xy[shown]!.x / W) * 100}%`, transform: `translateX(${xy[shown]!.x / W > 0.7 ? "-100%" : "8px"})` }}
            aria-hidden
          >
            <p className="font-semibold tabular-nums">{m.format(values[shown]!)}</p>
            <p className="text-muted-foreground">{full.format(new Date(points[shown]!.at))}</p>
          </div>
        )}
      </div>
    </div>
  );
}
