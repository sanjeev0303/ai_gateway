"use client";

import * as React from "react";
import { LogOut, Monitor, Moon, Search, Settings, Sun, User as UserIcon } from "lucide-react";

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
        <Button variant="ghost" size="icon" aria-label="Change theme">
          {resolvedTheme === "dark" ? (
            <Moon className="size-5" aria-hidden />
          ) : (
            <Sun className="size-5" aria-hidden />
          )}
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-36">
        <DropdownMenuLabel>Theme</DropdownMenuLabel>
        <DropdownMenuItem onClick={() => setTheme("light")}>
          <Sun className="size-4" aria-hidden /> Light
        </DropdownMenuItem>
        <DropdownMenuItem onClick={() => setTheme("dark")}>
          <Moon className="size-4" aria-hidden /> Dark
        </DropdownMenuItem>
        <DropdownMenuItem onClick={() => setTheme("system")}>
          <Monitor className="size-4" aria-hidden /> System
          {theme === "system" && <span className="ml-auto text-xs">active</span>}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

export function Topbar() {
  const { range, setRange } = useDateRange();
  const session = useSession();
  if (session.status !== "authenticated") return null;
  const { user, organization } = session.session;

  return (
    <header className="sticky top-0 z-30 flex h-14 shrink-0 items-center gap-2 border-b bg-background/80 px-4 backdrop-blur-sm sm:px-6">
      <MobileNav />

      <div className="flex min-w-0 items-center gap-2">
        <span className="truncate text-sm font-medium">
          {organization.name}
        </span>
        <StatusBadge tone={ORG_STATUS_TONE[organization.status]}>
          {ORG_STATUS_LABELS[organization.status]}
        </StatusBadge>
      </div>

      <div className="ml-auto flex items-center gap-1.5 sm:gap-2">
        <CommandPaletteTrigger />

        <Select value={range} onValueChange={setRange}>
          <SelectTrigger className="w-24" aria-label="Date range">
            <SelectValue placeholder="Range" />
          </SelectTrigger>
          <SelectContent>
            {DATE_RANGES.map((option) => (
              <SelectItem key={option} value={option}>
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
              className="rounded-full"
              aria-label="Account menu"
            >
              <span className="flex size-7 items-center justify-center rounded-full bg-primary/10 text-xs font-semibold text-primary">
                {initials(user.name)}
              </span>
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-56">
            <DropdownMenuLabel className="font-normal">
              <span className="block text-sm font-medium">{user.name}</span>
              <span className="block truncate text-xs text-muted-foreground">
                {user.email}
              </span>
            </DropdownMenuLabel>
            <DropdownMenuSeparator />
            <DropdownMenuItem>
              <UserIcon className="size-4" aria-hidden /> Profile
            </DropdownMenuItem>
            <DropdownMenuItem>
              <Settings className="size-4" aria-hidden /> Settings
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <form action={signOut}>
              <DropdownMenuItem asChild variant="destructive" className="w-full">
                <button type="submit">
                  <LogOut className="size-4" aria-hidden /> Sign out
                </button>
              </DropdownMenuItem>
            </form>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </header>
  );
}
