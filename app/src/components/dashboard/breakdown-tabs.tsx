"use client";

import Link from "next/link";
import { usePathname, useSearchParams } from "next/navigation";

import { cn } from "@/lib/utils";

const TABS = [
  { label: "Models", href: "/app/observe/models" },
  { label: "Providers", href: "/app/observe/providers" },
  { label: "Users", href: "/app/observe/users" },
  { label: "Teams", href: "/app/observe/teams" },
  { label: "Projects", href: "/app/observe/projects" },
];

/** The same traffic, cut five ways. Filters carry over when you switch the cut. */
export function BreakdownTabs() {
  const pathname = usePathname();
  const query = useSearchParams().toString();

  return (
    <nav aria-label="Break down by" className="-mx-1 overflow-x-auto px-1">
      <ul className="inline-flex h-9 items-center rounded-full border border-border bg-muted/60 p-[3px]">
        {TABS.map((tab) => {
          const active = pathname === tab.href || pathname.startsWith(`${tab.href}/`);
          return (
            <li key={tab.href} className="h-full">
              <Link
                href={query ? `${tab.href}?${query}` : tab.href}
                aria-current={active ? "page" : undefined}
                className={cn(
                  "flex h-full items-center rounded-full px-3.5 text-sm font-medium transition-[color,background-color] duration-150 focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none",
                  active ? "bg-card text-foreground shadow-[0_1px_2px_rgba(20,26,51,0.08)]" : "text-muted-foreground hover:text-foreground",
                )}
              >
                {tab.label}
              </Link>
            </li>
          );
        })}
      </ul>
    </nav>
  );
}
