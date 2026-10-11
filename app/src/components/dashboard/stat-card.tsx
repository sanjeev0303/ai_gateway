import Link from "next/link";
import type { LucideIcon } from "lucide-react";
import { ArrowDownRight, ArrowUpRight, Minus } from "lucide-react";

import { Sparkline } from "@/components/charts";
import { Card, CardContent } from "@/components/ui/card";
import { formatDelta } from "@/lib/format";
import { cn } from "@/lib/utils";
import type { StatusTone, TimeSeriesPoint, TrendDirection } from "@/types";

interface StatDelta {
  /** Fractional change, e.g. 0.12 for +12%. */
  fraction: number;
  direction: TrendDirection;
  /** Costs and error rates get worse as they rise; usage gets better. */
  higherIsWorse?: boolean;
  label?: string;
}

interface StatCardProps {
  label: string;
  value: string;
  hint?: string;
  delta?: StatDelta;
  sparkline?: readonly TimeSeriesPoint[];
  icon?: LucideIcon;
  /** Renders the value in the status colour. */
  tone?: StatusTone;
  href?: string;
}

const TONE_TEXT: Record<StatusTone, string> = {
  success: "text-emerald-600 dark:text-emerald-400",
  warning: "text-amber-600 dark:text-amber-400",
  danger: "text-rose-600 dark:text-rose-400",
  info: "text-sky-600 dark:text-sky-400",
  neutral: "text-foreground",
};

function DeltaBadge({ delta }: { delta: StatDelta }) {
  const Icon =
    delta.direction === "up" ? ArrowUpRight
    : delta.direction === "down" ? ArrowDownRight
    : Minus;

  let tone = "text-muted-foreground";
  if (delta.direction !== "flat") {
    const rising = delta.direction === "up";
    const bad = delta.higherIsWorse ? rising : !rising;
    tone = bad ? "text-rose-600 dark:text-rose-400" : "text-emerald-600 dark:text-emerald-400";
  }

  return (
    <span className={cn("inline-flex items-center gap-1 text-xs font-medium", tone)}>
      <Icon className="size-3.5" aria-hidden />
      {formatDelta(delta.fraction)}
      {delta.label && (
        <span className="font-normal text-muted-foreground">{delta.label}</span>
      )}
    </span>
  );
}

/** Headline number for dashboard rows; optionally a link and a trend sparkline. */
export function StatCard({
  label,
  value,
  hint,
  delta,
  sparkline,
  icon: Icon,
  tone = "neutral",
  href,
}: StatCardProps) {
  const body = (
    <CardContent className="flex items-start justify-between gap-4 px-5">
      <div className="min-w-0 space-y-1">
        <div className="flex items-center gap-2 text-sm text-muted-foreground">
          {Icon && <Icon className="size-4" aria-hidden />}
          <span className="truncate">{label}</span>
        </div>
        <p suppressHydrationWarning className={cn("text-2xl font-semibold tabular-nums", TONE_TEXT[tone])}>
          {value}
        </p>
        <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
          {delta && <DeltaBadge delta={delta} />}
          {hint && <span suppressHydrationWarning className="text-xs text-muted-foreground">{hint}</span>}
        </div>
      </div>
      {sparkline && sparkline.length > 1 && (
        <Sparkline data={sparkline} label={`${label} trend`} />
      )}
    </CardContent>
  );

  if (href) {
    return (
      <Card className="transition-colors hover:border-foreground/20">
        <Link href={href} className="block focus-visible:outline-none">
          {body}
        </Link>
      </Card>
    );
  }

  return <Card>{body}</Card>;
}
