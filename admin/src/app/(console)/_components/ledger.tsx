import Link from "next/link";
import type { LucideIcon } from "lucide-react";

import { GuillocheBand } from "@ai-governance/console-ui/brand/guilloche";
import { cn } from "@ai-governance/console-ui/lib/utils";

type Tone = "neutral" | "danger" | "warning" | "success";

const TONE_TEXT: Record<Tone, string> = {
  neutral: "text-foreground",
  danger: "text-rose-700 dark:text-rose-400",
  warning: "text-amber-700 dark:text-amber-400",
  success: "text-emerald-700 dark:text-emerald-400",
};

/**
 * The health ledger from DESIGN.md: one panel with a guilloche band, a title
 * row, and figures in ruled cells (a gap-px grid over the border colour).
 */
const LEDGER_COLUMNS = {
  3: "grid-cols-1 sm:grid-cols-3",
  4: "grid-cols-2 xl:grid-cols-4",
  6: "grid-cols-2 md:grid-cols-3 xl:grid-cols-6",
} as const;

export function Ledger({
  title,
  aside,
  columns = 6,
  children,
  className,
}: {
  title: string;
  aside?: React.ReactNode;
  /** How many cells the ledger holds, so no row is left half empty. */
  columns?: keyof typeof LEDGER_COLUMNS;
  children: React.ReactNode;
  className?: string;
}) {
  const id = `ledger-${title.toLowerCase().replace(/[^a-z0-9]+/g, "-")}`;
  return (
    <section aria-labelledby={id} className={cn("overflow-hidden rounded-2xl border border-border-strong bg-card", className)}>
      <GuillocheBand className="opacity-70" />
      <div className="flex items-center justify-between gap-3 px-5 pt-1 pb-3">
        <h2 id={id} className="text-[0.9375rem] font-semibold tracking-[-0.01em] text-foreground">
          {title}
        </h2>
        {aside}
      </div>
      <div className={cn("grid gap-px border-t border-border bg-border", LEDGER_COLUMNS[columns])}>
        {children}
      </div>
    </section>
  );
}

/** A figure with nothing behind it reads as a ghost numeral, per DESIGN.md. */
function isEmptyFigure(value: string): boolean {
  return value === "—" || /^\$?0(\.0+)?%?$/.test(value);
}

export function LedgerCell({
  label,
  value,
  detail,
  icon: Icon,
  tone = "neutral",
  href,
  emptyDetail = "Nothing in this range",
}: {
  label: string;
  value: string;
  detail?: string;
  /** What a zero means for this figure; not every figure is time-ranged. */
  emptyDetail?: string;
  icon: LucideIcon;
  tone?: Tone;
  href?: string;
}) {
  const empty = isEmptyFigure(value);
  const body = (
    <>
      <span className="flex items-center gap-2 text-[0.8125rem] font-medium text-muted-foreground">
        <Icon className="size-3.5 shrink-0 text-copper" aria-hidden />
        <span className="truncate">{label}</span>
      </span>
      <p className={cn("font-display text-[1.75rem] leading-none tabular-nums", empty ? "text-foreground/25" : TONE_TEXT[tone])}>
        {value}
      </p>
      <span className="truncate text-xs text-muted-foreground">{empty ? emptyDetail : detail}</span>
    </>
  );

  const cell = "flex min-w-0 flex-col justify-between gap-3 bg-card p-4";
  if (!href) return <div className={cell}>{body}</div>;
  return (
    <Link
      href={href}
      className={cn(
        cell,
        "transition-[background-color] duration-150 ease-(--ease-out-strong) hover:bg-accent/60 focus-visible:z-10 focus-visible:ring-3 focus-visible:ring-ring/40 focus-visible:outline-none",
      )}
    >
      {body}
    </Link>
  );
}

/** A flat panel with a title row, for registers and tables. */
export function Panel({
  title,
  description,
  action,
  children,
  className,
}: {
  title: string;
  description?: string;
  action?: React.ReactNode;
  children: React.ReactNode;
  className?: string;
}) {
  return (
    <section className={cn("flex min-w-0 flex-col rounded-2xl border border-border bg-card", className)}>
      <div className="flex flex-wrap items-start justify-between gap-3 px-5 pt-4 pb-3">
        <div className="flex min-w-0 flex-col gap-1">
          <h2 className="text-[0.9375rem] font-semibold tracking-[-0.01em] text-foreground">{title}</h2>
          {description && <p className="max-w-[60ch] text-sm text-muted-foreground">{description}</p>}
        </div>
        {action}
      </div>
      <div className="min-w-0 flex-1 border-t border-border">{children}</div>
    </section>
  );
}

/** A ruled register: rows divided by hairlines, never separate cards. */
export function Register({ children }: { children: React.ReactNode }) {
  return <ul className="divide-y divide-border">{children}</ul>;
}

export function RegisterRow({
  primary,
  secondary,
  value,
  href,
}: {
  primary: React.ReactNode;
  secondary?: React.ReactNode;
  value?: React.ReactNode;
  href?: string;
}) {
  const content = (
    <>
      <span className="flex min-w-0 flex-col gap-0.5">
        <span className="truncate text-sm font-medium text-foreground">{primary}</span>
        {secondary && <span className="truncate text-xs text-muted-foreground">{secondary}</span>}
      </span>
      {value !== undefined && <span className="shrink-0 text-sm text-foreground tabular-nums">{value}</span>}
    </>
  );
  const row = "flex items-center justify-between gap-4 px-5 py-3";
  return (
    <li>
      {href ? (
        <Link
          href={href}
          className={cn(
            row,
            "transition-[background-color] duration-150 ease-(--ease-out-strong) hover:bg-accent/60 focus-visible:ring-3 focus-visible:ring-ring/40 focus-visible:outline-none",
          )}
        >
          {content}
        </Link>
      ) : (
        <div className={row}>{content}</div>
      )}
    </li>
  );
}

export function RegisterEmpty({ children }: { children: React.ReactNode }) {
  return <p className="px-5 py-8 text-center text-sm text-muted-foreground">{children}</p>;
}
