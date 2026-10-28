"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { ArrowDown, ArrowUp, ChevronLeft, ChevronRight } from "lucide-react";

import { StatusBadge } from "@/components/dashboard/status-badge";
import { EmptyState } from "@/components/dashboard/state";
import { Button } from "@/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  formatCompact,
  formatDate,
  formatDateTime,
  formatMs,
  formatNumber,
  formatPercent,
  formatRelative,
  formatUsd,
  formatUsdExact,
} from "@/lib/format";
import type { StatusTone } from "@/types";
import { cn } from "@/lib/utils";

/**
 * How a cell's primary value is rendered. Every entry is a string so a Server
 * Component can build columns as plain, serializable data.
 */
export type CellFormat =
  | "text"
  | "number"
  | "compact"
  | "usd"
  | "usdExact"
  | "percent"
  | "percentRaw"
  | "ms"
  | "date"
  | "dateTime"
  | "relative";

const FORMATTERS: Record<CellFormat, (value: unknown) => string> = {
  text: (value) =>
    value === null || value === undefined || value === "" ? "—" : String(value),
  number: (value) => formatNumber(Number(value)),
  compact: (value) => formatCompact(Number(value)),
  usd: (value) => formatUsd(Number(value)),
  usdExact: (value) => formatUsdExact(Number(value)),
  percent: (value) => formatPercent(Number(value)),
  percentRaw: (value) => formatPercent(Number(value) / 100),
  ms: (value) => formatMs(Number(value)),
  date: (value) => formatDate(String(value)),
  dateTime: (value) => formatDateTime(String(value)),
  relative: (value) => formatRelative(String(value)),
};

const TABULAR: ReadonlySet<CellFormat> = new Set<CellFormat>([
  "number",
  "compact",
  "usd",
  "usdExact",
  "ms",
  "percent",
  "percentRaw",
]);

type Field<T> = Extract<keyof T, string>;

/**
 * A column is declarative: it names row fields and says how to format them, so
 * columns stay serializable and pages stay Server Components. All domain
 * shaping — labels, tones, derived numbers — happens server side, and the table
 * receives plain view-model rows.
 */
export interface Column<T> {
  key: string;
  header: string;
  /** Row field rendered as the main cell value. */
  primary: Field<T>;
  /** Optional muted second line under the primary value. */
  secondary?: Field<T>;
  format?: CellFormat;
  align?: "left" | "right";
  /** Row field holding a precomputed tone; renders a badge instead of text. */
  tone?: Field<T>;
  /** Row field holding an href; wraps the primary value in a link. */
  href?: Field<T>;
  /** Row field holding a value against `meterMax`; renders a thin meter bar. */
  meter?: Field<T>;
  meterMax?: number;
  /** Field to sort on; defaults to `primary`. */
  sort?: Field<T>;
  /** Optional custom cell renderer. */
  render?: (row: T) => React.ReactNode;
}

interface DataTableProps<T> {
  columns: readonly Column<T>[];
  rows: readonly T[];
  /** Row field holding a stable unique id. */
  rowKey: Field<T>;
  pageSize?: number;
  emptyMessage?: string;
  /** Slot above the table for filters or a result count. */
  toolbar?: React.ReactNode;
  /** Optional client component for rendering actions cell. Receives row as prop. */
  ActionsComponent?: React.ComponentType<{ row: T } & Record<string, unknown>>;
  loading?: boolean;
  /** If true, pagination is managed by the caller. */
  serverPaginated?: boolean;
  /** True if there is a next page (for serverPaginated). */
  hasMore?: boolean;
  /** Caller-provided page number (for serverPaginated, 0-indexed). */
  currentPage?: number;
  nextHref?: string;
  prevHref?: string;
}

function cellText(row: Record<string, unknown>, field: string, format: CellFormat): string {
  return FORMATTERS[format](row[field]);
}

/**
 * Generic sortable, paginated table. Every `/app` list page renders through this
 * so column density, alignment, and pager behaviour stay identical.
 */
export function DataTable<T>({
  columns,
  rows,
  rowKey,
  pageSize = 10,
  emptyMessage = "No records found.",
  toolbar,
  ActionsComponent,
  serverPaginated = false,
  hasMore = false,
  currentPage: serverPage = 0,
  nextHref,
  prevHref,
}: DataTableProps<T>) {
  const router = useRouter();
  const [sortKey, setSortKey] = React.useState<string | null>(null);
  const [ascending, setAscending] = React.useState(true);
  const [localPage, setLocalPage] = React.useState(0);

  const sorted = React.useMemo(() => {
    const column = columns.find((entry) => entry.key === sortKey);
    if (!column) return rows;
    const field = (column.sort ?? column.primary) as string;
    const copy = [...rows] as Record<string, unknown>[];
    copy.sort((a, b) => {
      const left = a[field];
      const right = b[field];
      if (left === right) return 0;
      let order: number;
      if (typeof left === "number" && typeof right === "number") {
        order = left - right;
      } else {
        order = String(left).localeCompare(String(right));
      }
      return ascending ? order : -order;
    });
    return copy as T[];
  }, [rows, columns, sortKey, ascending]);

  const pageCount = serverPaginated ? Math.max(1, serverPage + (hasMore ? 2 : 1)) : Math.max(1, Math.ceil(sorted.length / pageSize));
  const currentPage = serverPaginated ? serverPage : Math.min(localPage, pageCount - 1);
  const visible = serverPaginated ? sorted : sorted.slice(
    currentPage * pageSize,
    currentPage * pageSize + pageSize,
  );

  const toggleSort = (column: Column<T>) => {
    if (column.key === sortKey) {
      setAscending((value) => !value);
      return;
    }
    setSortKey(column.key);
    setAscending(true);
  };

  if (rows.length === 0) {
    return (
      <div className="space-y-3">
        {toolbar}
        <EmptyState title={emptyMessage} />
      </div>
    );
  }

  return (
    <div className="space-y-3">
      {toolbar}

      <div className="overflow-x-auto rounded-lg border">
        <Table>
          <TableHeader>
            <TableRow>
              {columns.map((column) => {
                const active = column.key === sortKey;
                return (
                  <TableHead
                    key={column.key}
                    className={cn(
                      column.align === "right" && "text-right",
                      column.meter && "w-44",
                    )}
                  >
                    <button
                      type="button"
                      onClick={() => toggleSort(column)}
                      className={cn(
                        "inline-flex items-center gap-1.5 rounded-sm transition-colors hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring text-sm font-semibold",
                        active && "text-foreground",
                        column.align === "right" && "flex-row-reverse",
                      )}
                    >
                      {column.header}
                      <span className={cn("transition-opacity", !active && "opacity-0")}>
                        {ascending ? (
                          <ArrowUp className="size-3.5" aria-hidden />
                        ) : (
                          <ArrowDown className="size-3.5" aria-hidden />
                        )}
                      </span>
                    </button>
                  </TableHead>
                );
              })}
              {ActionsComponent && (
                <TableHead className="w-12 text-right">
                  <span className="sr-only">Actions</span>
                </TableHead>
              )}
            </TableRow>
          </TableHeader>

          <TableBody>
            {visible.map((row) => {
              const record = row as Record<string, unknown>;
              return (
                <TableRow key={String(record[rowKey as string])}>
                  {columns.map((column) => {
                    const alignRight = column.align === "right";
                    const format = column.format ?? "text";

                    if (column.tone) {
                      return (
                        <TableCell key={column.key} suppressHydrationWarning>
                          <StatusBadge tone={record[column.tone] as StatusTone}>
                            {cellText(record, column.primary, format)}
                          </StatusBadge>
                        </TableCell>
                      );
                    }

                    if (column.render) {
                      return (
                        <TableCell key={column.key} suppressHydrationWarning className={cn(alignRight && "text-right")}>
                          {column.render(row)}
                        </TableCell>
                      );
                    }

                    const primaryText = cellText(record, column.primary, format);
                    const href = column.href ? record[column.href] : undefined;
                    const secondaryText = column.secondary
                      ? cellText(record, column.secondary, "text")
                      : null;
                    const meterValue = column.meter
                      ? Number(record[column.meter])
                      : null;

                    return (
                      <TableCell
                        key={column.key}
                        suppressHydrationWarning
                        className={cn(alignRight && "text-right")}
                      >
                        <div
                          className={cn(
                            "min-w-0",
                            alignRight && "flex flex-col items-end",
                          )}
                        >
                          {href ? (
                            <a
                              href={String(href)}
                              suppressHydrationWarning
                              className="font-medium underline-offset-4 hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                            >
                              {primaryText}
                            </a>
                          ) : (
                            <span
                              suppressHydrationWarning
                              className={cn(
                                "block",
                                TABULAR.has(format) && "tabular-nums",
                              )}
                            >
                              {primaryText}
                            </span>
                          )}
                          {secondaryText && secondaryText !== "—" && (
                            <span className="block truncate text-sm text-muted-foreground/90">
                              {secondaryText}
                            </span>
                          )}
                          {meterValue !== null && (
                            <span
                              aria-hidden
                              className="mt-1.5 block h-1.5 w-full overflow-hidden rounded-full bg-muted"
                            >
                              <span
                                className="block h-full rounded-full bg-primary"
                                style={{
                                  width: `${Math.min(
                                    100,
                                    Math.max(0, (meterValue / (column.meterMax ?? 100)) * 100),
                                  )}%`,
                                }}
                              />
                            </span>
                          )}
                        </div>
                      </TableCell>
                    );
                  })}
                  {ActionsComponent && (
                    <TableCell className="text-right">
                      <ActionsComponent row={row} />
                    </TableCell>
                  )}
                </TableRow>
              );
            })}
          </TableBody>
        </Table>
      </div>

      {pageCount > 1 && (
        <div className="flex flex-wrap items-center justify-between gap-2 text-sm text-muted-foreground">
          <span className="tabular-nums">
            {serverPaginated ? (
              <>Page {currentPage + 1}</>
            ) : (
              <>{currentPage * pageSize + 1}–{Math.min(sorted.length, (currentPage + 1) * pageSize)} of {sorted.length}</>
            )}
          </span>
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              disabled={serverPaginated ? !prevHref : currentPage === 0}
              onClick={() => {
                if (serverPaginated && prevHref) {
                  router.push(prevHref);
                } else if (!serverPaginated) {
                  setLocalPage(Math.max(0, currentPage - 1));
                }
              }}
            >
              <ChevronLeft className="size-4" aria-hidden /> Previous
            </Button>
            {!serverPaginated && (
              <span className="tabular-nums">
                Page {currentPage + 1} of {pageCount}
              </span>
            )}
            <Button
              variant="outline"
              size="sm"
              disabled={serverPaginated ? !nextHref : currentPage >= pageCount - 1}
              onClick={() => {
                if (serverPaginated && nextHref) {
                  router.push(nextHref);
                } else if (!serverPaginated) {
                  setLocalPage(Math.min(pageCount - 1, currentPage + 1));
                }
              }}
            >
              Next <ChevronRight className="size-4" aria-hidden />
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}
