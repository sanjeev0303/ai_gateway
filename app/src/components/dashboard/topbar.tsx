"use client";

import * as React from "react";
import { LogOut, Monitor, Moon, Search, Settings, Sun, User as UserIcon } from "lucide-react";

import { MobileNav } from "./mobile-nav";
import { StatusBadge } from "./status-badge";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useTheme } from "@/hooks/use-theme";
import { DATE_RANGES } from "@/lib/constants";
import { currentUser, organization } from "@/lib/mock-data";
import { initials } from "@/lib/format";
import { ORG_STATUS_TONE, PLAN_LABELS } from "@/types";

/** Sun/moon/system switch. `useTheme` only mutates the DOM after mount. */
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

interface TopbarProps {
  /** Default range; the picker is uncontrolled and page-local. */
  defaultRange?: (typeof DATE_RANGES)[number];
}

export function Topbar({ defaultRange = "7d" }: TopbarProps) {
  const [range, setRange] = React.useState<string>(defaultRange);

  return (
    <header className="sticky top-0 z-30 flex h-14 shrink-0 items-center gap-2 border-b bg-background/80 px-3 backdrop-blur-sm sm:px-4">
      <MobileNav />

      <div className="flex min-w-0 items-center gap-2">
        <span className="truncate text-sm font-medium">
          {organization.name}
        </span>
        <StatusBadge tone={ORG_STATUS_TONE[organization.status]}>
          {PLAN_LABELS[organization.plan]}
        </StatusBadge>
      </div>

      <div className="ml-auto flex items-center gap-1.5 sm:gap-2">
        <div className="relative hidden md:block">
          <Search
            className="pointer-events-none absolute left-2.5 top-1/2 size-4 -translate-y-1/2 text-muted-foreground"
            aria-hidden
          />
          <Input
            type="search"
            placeholder="Search requests, users, policies"
            aria-label="Search"
            className="w-64 pl-8"
          />
        </div>

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
                {initials(currentUser.name)}
              </span>
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-56">
            <DropdownMenuLabel className="font-normal">
              <span className="block text-sm font-medium">{currentUser.name}</span>
              <span className="block truncate text-xs text-muted-foreground">
                {currentUser.email}
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
            <DropdownMenuItem variant="destructive">
              <LogOut className="size-4" aria-hidden /> Sign out
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </header>
  );
}
