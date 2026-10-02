import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import type { StatusTone } from "@/types";

/**
 * Domain types already export their own `*_TONE` maps, so callers pass a tone
 * in rather than a status string. That keeps the colour vocabulary in one place.
 */
const TONE_CLASS: Record<StatusTone, string> = {
  success:
    "bg-emerald-500/10 text-emerald-700 ring-emerald-500/20 dark:text-emerald-400",
  warning:
    "bg-amber-500/10 text-amber-700 ring-amber-500/20 dark:text-amber-400",
  danger: "bg-rose-500/10 text-rose-700 ring-rose-500/20 dark:text-rose-400",
  info: "bg-sky-500/10 text-sky-700 ring-sky-500/20 dark:text-sky-400",
  neutral: "bg-muted text-muted-foreground ring-border",
};

export function toneClass(tone: StatusTone): string {
  return TONE_CLASS[tone];
}

interface StatusBadgeProps {
  tone: StatusTone;
  children: React.ReactNode;
  className?: string;
}

export function StatusBadge({ tone, children, className }: StatusBadgeProps) {
  return (
    <Badge
      variant="outline"
      className={cn("ring-1 ring-inset", TONE_CLASS[tone], className)}
    >
      {children}
    </Badge>
  );
}
