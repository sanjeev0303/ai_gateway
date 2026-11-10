import Link from "next/link";
import type { LucideIcon } from "lucide-react";
import { ArrowDownRight, ArrowUpRight, Minus } from "lucide-react";

import { Sparkline } from "@/components/charts";
import { Card, CardContent } from "@/components/ui/card";
import { formatDelta } from "@ai-governance/console-ui/lib/format";
import { cn } from "@ai-governance/console-ui/lib/utils";
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
  neumorphic?: boolean;
  className?: string;
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
    <span className={cn("inline-flex items-center gap-1 text-[0.8125rem] font-medium tabular-nums", tone)}>
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
  neumorphic = false,
  className,
}: StatCardProps) {
  const body = (
    <CardContent className="flex items-start justify-between gap-4 px-5 py-1">
      <div className="flex min-w-0 items-start gap-3">
        {Icon && (
          <span className="mt-0.5 flex size-9 shrink-0 items-center justify-center rounded-xl border border-border bg-accent/70 text-copper">
            <Icon className="size-4" aria-hidden />
          </span>
        )}
        <div className="flex min-w-0 flex-col gap-1">
          <span className="truncate text-[0.8125rem] font-medium text-muted-foreground">{label}</span>
          <p
            suppressHydrationWarning
            data-slot="stat-value"
            className={cn("text-[1.625rem] leading-tight font-semibold tracking-[-0.02em] tabular-nums", TONE_TEXT[tone])}
          >
            {value}
          </p>
          <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
            {delta && <DeltaBadge delta={delta} />}
            {hint && <span suppressHydrationWarning className="text-[0.8125rem] text-muted-foreground">{hint}</span>}
          </div>
        </div>
      </div>
      {sparkline && sparkline.length > 1 && (
        <div className="hidden pt-1 sm:block">
          <Sparkline data={sparkline} label={`${label} trend`} />
        </div>
      )}
    </CardContent>
  );

  const cardClasses = cn(
    "rounded-2xl transition-[border-color,box-shadow] duration-200",
    neumorphic ? "neu-card border-none" : "border-border bg-card",
    className,
  );

  if (href) {
    return (
      <Card className={cn(cardClasses, "cursor-pointer hover:border-border-strong hover:shadow-(--shadow-float)")}>
        <Link href={href} className="block focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring rounded-2xl">
          {body}
        </Link>
      </Card>
    );
  }

  return <Card className={cardClasses}>{body}</Card>;
}
