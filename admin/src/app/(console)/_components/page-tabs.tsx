import Link from "next/link";

import { cn } from "@ai-governance/console-ui/lib/utils";

interface PageTab {
  value: string;
  label: string;
  href: string;
}

/**
 * Tabs held in the URL, so a view can be linked and survives a reload. Drawn
 * like the date-range pills used across the console.
 */
export function PageTabs({ tabs, current, label }: { tabs: PageTab[]; current: string; label: string }) {
  return (
    <nav aria-label={label} className="liquid-glass inline-flex h-9 max-w-full items-center gap-0.5 overflow-x-auto rounded-full border p-[3px]">
      {tabs.map((tab) => {
        const active = tab.value === current;
        return (
          <Link
            key={tab.value}
            href={tab.href}
            aria-current={active ? "page" : undefined}
            className={cn(
              "inline-flex h-full shrink-0 items-center rounded-full px-3 text-[0.8125rem] font-medium transition-[background-color,color] duration-150 ease-(--ease-out-strong) focus-visible:ring-2 focus-visible:ring-ring/40 focus-visible:outline-none",
              active ? "bg-primary text-primary-foreground" : "text-muted-foreground hover:text-foreground",
            )}
          >
            {tab.label}
          </Link>
        );
      })}
    </nav>
  );
}
