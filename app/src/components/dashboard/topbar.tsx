"use client";

import * as React from "react";
import { LogOut, Monitor, Moon, Settings, Sun, User as UserIcon, ChevronRight } from "lucide-react";
import { usePathname } from "next/navigation";

import { MobileNav } from "./mobile-nav";
import { StatusBadge } from "./status-badge";
import { signOut } from "@/app/(auth)/signin/actions";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useTheme } from "next-themes";
import { DATE_RANGES } from "@/lib/constants";
import { initials } from "@/lib/format";
import { useSession } from "@/lib/session-context";
import { ORG_STATUS_LABELS, ORG_STATUS_TONE } from "@/types";
import { CommandPaletteTrigger } from "./command-palette";
import { useDateRange } from "@/hooks/use-date-range";

function ThemeToggle() {
  const { theme, resolvedTheme, setTheme } = useTheme();

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="icon" className="size-8 rounded-full" aria-label="Change theme">
          {resolvedTheme === "dark" ? (
            <Moon className="size-4" aria-hidden />
          ) : (
            <Sun className="size-4" aria-hidden />
          )}
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-36 glass">
        <DropdownMenuLabel className="text-xs">Theme</DropdownMenuLabel>
        <DropdownMenuItem onClick={() => setTheme("light")} className="text-xs">
          <Sun className="mr-2 size-3.5" aria-hidden /> Light
        </DropdownMenuItem>
        <DropdownMenuItem onClick={() => setTheme("dark")} className="text-xs">
          <Moon className="mr-2 size-3.5" aria-hidden /> Dark
        </DropdownMenuItem>
        <DropdownMenuItem onClick={() => setTheme("system")} className="text-xs">
          <Monitor className="mr-2 size-3.5" aria-hidden /> System
          {theme === "system" && <span className="ml-auto text-[10px] text-muted-foreground">active</span>}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function formatSegment(seg: string): string {
  return seg
    .replace(/-/g, " ")
    .replace(/\b\w/g, (c) => c.toUpperCase());
}

export function Topbar() {
  const pathname = usePathname();
  const { range, setRange } = useDateRange();
  const session = useSession();
  if (session.status !== "authenticated") return null;
  const { user, organization } = session.session;

  const segments = pathname.replace(/^\/app\/?/, "").split("/").filter(Boolean);
  const breadcrumbs = segments.length > 0 ? segments.map(formatSegment) : ["Overview"];

  return (
    <header className="sticky top-0 z-30 flex h-14 shrink-0 items-center gap-3 border-b border-border/40 bg-background/80 px-4 backdrop-blur-md sm:px-6">
      <MobileNav />

      {/* Breadcrumb path */}
      <nav aria-label="Breadcrumb" className="hidden sm:flex items-center gap-1.5 text-xs">
        <span className="font-semibold text-foreground/80">{organization.name}</span>
        {breadcrumbs.map((crumb, idx) => (
          <React.Fragment key={crumb}>
            <ChevronRight className="size-3 text-muted-foreground/60" aria-hidden />
            <span
              className={
                idx === breadcrumbs.length - 1
                  ? "font-semibold text-foreground"
                  : "text-muted-foreground"
              }
            >
              {crumb}
            </span>
          </React.Fragment>
        ))}
      </nav>

      <div className="flex sm:hidden min-w-0 items-center gap-2">
        <span className="truncate text-sm font-semibold">{organization.name}</span>
      </div>

      <div className="hidden md:inline-flex">
        <StatusBadge tone={ORG_STATUS_TONE[organization.status]}>
          {ORG_STATUS_LABELS[organization.status]}
        </StatusBadge>
      </div>

      <div className="ml-auto flex items-center gap-1.5 sm:gap-2">
        <CommandPaletteTrigger />

        <Select value={range} onValueChange={setRange}>
          <SelectTrigger className="h-8 w-24 text-xs rounded-lg border-border/60" aria-label="Date range">
            <SelectValue placeholder="Range" />
          </SelectTrigger>
          <SelectContent className="glass">
            {DATE_RANGES.map((option) => (
              <SelectItem key={option} value={option} className="text-xs">
                {option}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>

        <ThemeToggle />

        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              variant="ghost"
              size="icon"
              className="size-8 rounded-full ring-2 ring-primary/20 hover:ring-primary/40 transition-all"
              aria-label="Account menu"
            >
              <span className="flex size-7 items-center justify-center rounded-full bg-primary/10 text-xs font-bold text-primary">
                {initials(user.name)}
              </span>
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-56 glass">
            <DropdownMenuLabel className="font-normal">
              <span className="block text-sm font-semibold">{user.name}</span>
              <span className="block truncate text-xs text-muted-foreground">
                {user.email}
              </span>
            </DropdownMenuLabel>
            <DropdownMenuSeparator />
            <DropdownMenuItem className="text-xs">
              <UserIcon className="mr-2 size-4" aria-hidden /> Profile
            </DropdownMenuItem>
            <DropdownMenuItem className="text-xs">
              <Settings className="mr-2 size-4" aria-hidden /> Settings
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
