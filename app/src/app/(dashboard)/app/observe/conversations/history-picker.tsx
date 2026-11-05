"use client";

import { usePathname, useRouter, useSearchParams } from "next/navigation";
import type { RailEmployee } from "./history-rail";

const SELECT =
  "h-10 w-full min-w-0 rounded-md border border-input bg-background px-3 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring";

/** The rail on narrow screens: pick an employee, then one of their keys. */
export function HistoryPicker({ employees }: { employees: readonly RailEmployee[] }) {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const userId = searchParams.get("user_id") ?? "";
  const keyId = searchParams.get("credential_id") ?? "";
  const employee = employees.find((e) => e.id === userId);

  const go = (nextUser: string, nextKey: string) => {
    const params = new URLSearchParams(searchParams.toString());
    params.delete("offset");
    if (nextUser) params.set("user_id", nextUser);
    else params.delete("user_id");
    if (nextKey) params.set("credential_id", nextKey);
    else params.delete("credential_id");
    const query = params.toString();
    router.push(query ? `${pathname}?${query}` : pathname, { scroll: false });
  };

  return (
    <div className="grid gap-2 sm:grid-cols-2">
      <label className="flex flex-col gap-1 text-xs font-medium text-muted-foreground">
        Employee
        <select className={SELECT} value={userId} onChange={(e) => go(e.target.value, "")}>
          <option value="">All employees</option>
          {employees.map((e) => (
            <option key={e.id} value={e.id}>
              {e.name} ({e.total})
            </option>
          ))}
        </select>
      </label>
      <label className="flex flex-col gap-1 text-xs font-medium text-muted-foreground">
        Gateway key
        <select className={SELECT} value={keyId} disabled={!employee} onChange={(e) => go(userId, e.target.value)}>
          <option value="">All keys</option>
          {employee?.keys.map((k) => (
            <option key={k.id ?? "none"} value={k.id ?? "none"}>
              {k.label}
              {k.prefix ? ` · ${k.prefix}` : ""} ({k.count})
            </option>
          ))}
        </select>
      </label>
    </div>
  );
}
