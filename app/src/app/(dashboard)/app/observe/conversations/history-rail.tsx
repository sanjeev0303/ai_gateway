import Link from "next/link";
import { cn } from "@/lib/utils";
import { formatNumber } from "@/lib/format";

export interface RailKey {
  /** Null groups the conversations that recorded no gateway key. */
  id: string | null;
  label: string;
  prefix?: string;
  /** The key's status when it is no longer active (revoked, expired). */
  inactive?: string;
  count: number;
}

export interface RailEmployee {
  id: string;
  name: string;
  total: number;
  keys: RailKey[];
}

const ROW = "flex min-w-0 items-center gap-2 rounded-lg px-2.5 py-1.5 text-sm transition-colors hover:bg-accent";
const ACTIVE = "bg-primary/10 text-primary hover:bg-primary/10";

/**
 * Employees, each opening to the gateway keys their conversations came through.
 * Plain links, so the selection lives in the URL and survives reloads.
 */
export function HistoryRail({
  employees,
  userId,
  keyId,
  href,
}: {
  employees: readonly RailEmployee[];
  userId?: string;
  keyId?: string;
  href: (userId?: string, keyId?: string | null) => string;
}) {
  const allCount = employees.reduce((sum, e) => sum + e.total, 0);
  return (
    <nav aria-label="Employees and their gateway keys" className="flex flex-col gap-1">
      <Link href={href()} aria-current={!userId ? "page" : undefined} className={cn(ROW, "font-medium", !userId && ACTIVE)}>
        <span className="min-w-0 flex-1 truncate">All employees</span>
        <span className="text-xs text-muted-foreground tabular-nums">{formatNumber(allCount)}</span>
      </Link>

      {employees.length === 0 && <p className="px-2.5 py-2 text-sm text-muted-foreground">No conversations yet.</p>}

      {employees.map((employee) => {
        const open = employee.id === userId;
        return (
          <details key={employee.id} open={open} className="group">
            <summary
              className={cn(
                ROW,
                "cursor-pointer list-none font-medium [&::-webkit-details-marker]:hidden",
                open && !keyId && ACTIVE,
              )}
            >
              <span aria-hidden className="text-xs text-muted-foreground transition-transform group-open:rotate-90">
                ▸
              </span>
              <span className="min-w-0 flex-1 truncate">{employee.name}</span>
              <span className="text-xs text-muted-foreground tabular-nums">{formatNumber(employee.total)}</span>
            </summary>
            <ul className="mt-0.5 mb-1 ml-4 flex flex-col gap-0.5 border-l border-border pl-2">
              <li>
                <Link
                  href={href(employee.id)}
                  aria-current={open && !keyId ? "page" : undefined}
                  className={cn(ROW, "text-muted-foreground", open && !keyId && ACTIVE)}
                >
                  <span className="min-w-0 flex-1 truncate">All keys</span>
                </Link>
              </li>
              {employee.keys.map((key) => {
                const active = open && keyId === (key.id ?? "none");
                return (
                  <li key={key.id ?? "none"}>
                    <Link
                      href={href(employee.id, key.id)}
                      aria-current={active ? "page" : undefined}
                      title={key.prefix ? `${key.label} · ${key.prefix}` : key.label}
                      className={cn(ROW, active && ACTIVE)}
                    >
                      <span className="flex min-w-0 flex-1 flex-col">
                        <span className="truncate">
                          {key.label}
                          {key.inactive && <span className="ml-1.5 text-xs text-muted-foreground">{key.inactive}</span>}
                        </span>
                        {key.prefix && <span className="truncate font-mono text-xs text-muted-foreground">{key.prefix}</span>}
                      </span>
                      <span className="text-xs text-muted-foreground tabular-nums">{formatNumber(key.count)}</span>
                    </Link>
                  </li>
                );
              })}
            </ul>
          </details>
        );
      })}
    </nav>
  );
}
