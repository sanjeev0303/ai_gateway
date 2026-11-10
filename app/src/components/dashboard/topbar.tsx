"use client";

import * as React from "react";
import Link from "next/link";
import { LogOut, Monitor, Moon, Settings, Sun, ChevronRight } from "lucide-react";
import { usePathname } from "next/navigation";

import { DASHBOARD_NAV } from "@/lib/constants";
import { MobileNav } from "./mobile-nav";
import { StatusBadge } from "@ai-governance/console-ui/dashboard/status-badge";
import { signOut } from "@/app/(auth)/signin/actions";
import { Button } from "@ai-governance/console-ui/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@ai-governance/console-ui/ui/dropdown-menu";
import { useTheme } from "next-themes";
import { initials } from "@ai-governance/console-ui/lib/format";
import { useSession } from "@/lib/session-context";
import { ORG_STATUS_LABELS, ORG_STATUS_TONE } from "@/types";
import { CommandPaletteTrigger } from "./command-palette";

function ThemeToggle() {
  const { theme, resolvedTheme, setTheme } = useTheme();

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="outline" size="icon" className="bg-card/80" aria-label="Change theme">
          {resolvedTheme === "dark" ? (
            <Moon className="size-4" aria-hidden />
          ) : (
            <Sun className="size-4" aria-hidden />
          )}
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-40">
        <DropdownMenuLabel className="text-xs">Theme</DropdownMenuLabel>
        <DropdownMenuItem onClick={() => setTheme("light")} className="text-sm">
          <Sun className="mr-2 size-4" aria-hidden /> Light
        </DropdownMenuItem>
        <DropdownMenuItem onClick={() => setTheme("dark")} className="text-sm">
          <Moon className="mr-2 size-4" aria-hidden /> Dark
        </DropdownMenuItem>
        <DropdownMenuItem onClick={() => setTheme("system")} className="text-sm">
          <Monitor className="mr-2 size-4" aria-hidden /> System
          {theme === "system" && <span className="ml-auto text-xs text-muted-foreground">active</span>}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

/** Sidebar labels by path, so a breadcrumb reads "CLI" and "API reference", not "Cli" and "Api Docs". */
const NAV_LABELS = new Map<string, string>(
  DASHBOARD_NAV.flatMap((section) => [
    [section.href, section.title] as const,
    ...section.children.map((child) => [child.href, child.label] as const),
  ]),
);

function formatSegment(seg: string, index: number, segments: string[]): string {
  const href = `/app/${segments.slice(0, index + 1).join("/")}`;
  const known = NAV_LABELS.get(href);
  if (known) return known;
  // A section folder ("developer") takes its section's title.
  const section = index === 0 ? DASHBOARD_NAV.find((s) => s.href.split("/")[2] === seg) : undefined;
  if (section) return section.title;
  return seg.replace(/-/g, " ").replace(/\b\w/g, (c) => c.toUpperCase());
}

export function Topbar() {
  const pathname = usePathname();
  const session = useSession();
  if (session.status !== "authenticated") return null;
  const { user, organization } = session.session;

  const segments = pathname.replace(/^\/app\/?/, "").split("/").filter(Boolean);
  const breadcrumbs = segments.length > 0 ? segments.map(formatSegment) : ["Overview"];

  return (
    <header className="liquid-glass sticky top-3 z-30 mx-3 mt-3 flex h-14 shrink-0 items-center gap-3 rounded-2xl border px-4 sm:mx-4 sm:px-5 lg:mx-6">
      <MobileNav />

      {/* Where you are: the page, then the path to it */}
      <div className="hidden min-w-0 sm:flex">
        <nav aria-label="Breadcrumb" className="flex items-center gap-1.5 text-[0.8125rem] text-muted-foreground">
          <span className="truncate">{organization.name}</span>
          {breadcrumbs.map((crumb, idx) => (
            <React.Fragment key={crumb}>
              <ChevronRight className="size-3.5 shrink-0 opacity-50" aria-hidden />
              <span
                aria-current={idx === breadcrumbs.length - 1 ? "page" : undefined}
                className={idx === breadcrumbs.length - 1 ? "truncate font-medium text-foreground" : "truncate"}
              >
                {crumb}
              </span>
            </React.Fragment>
          ))}
        </nav>
      </div>

      <div className="flex sm:hidden min-w-0 items-center gap-2">
        <span className="truncate text-sm font-semibold">{organization.name}</span>
      </div>

      {organization.status !== "active" && (
        <div className="hidden md:inline-flex">
          <StatusBadge tone={ORG_STATUS_TONE[organization.status]}>
            {ORG_STATUS_LABELS[organization.status]}
          </StatusBadge>
        </div>
      )}

      <div className="ml-auto flex items-center gap-1.5 sm:gap-2">
        <CommandPaletteTrigger />

        <ThemeToggle />

        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              variant="ghost"
              size="icon"
              className="size-9 rounded-full p-0"
              aria-label="Account menu"
            >
              <span className="flex size-9 items-center justify-center rounded-full border border-border bg-accent text-xs font-semibold text-primary transition-[border-color] duration-150 hover:border-border-strong">
                {initials(user.name)}
              </span>
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-60">
            <DropdownMenuLabel className="font-normal">
              <span className="block text-sm font-semibold">{user.name}</span>
              <span className="block truncate text-xs text-muted-foreground">
                {user.email}
              </span>
            </DropdownMenuLabel>
            <DropdownMenuSeparator />
            <DropdownMenuItem asChild className="text-xs">
              <Link href="/app/organization/settings">
                <Settings className="mr-2 size-4" aria-hidden /> Workspace settings
              </Link>
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <form action={signOut}>
              <DropdownMenuItem asChild variant="destructive" className="w-full text-xs">
                <button type="submit">
                  <LogOut className="mr-2 size-4" aria-hidden /> Sign out
                </button>
              </DropdownMenuItem>
            </form>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </header>
  );
}
