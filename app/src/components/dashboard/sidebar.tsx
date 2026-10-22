"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  Building2,
  ChevronDown,
  ChevronLeft,
  ChevronRight,
  LifeBuoy,
  Plus,
  Settings,
} from "lucide-react";

import { Logo } from "@/components/logo";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { DASHBOARD_NAV } from "@/lib/constants";
import { initials } from "@/lib/format";
import { useSession } from "@/lib/session-context";
import { cn } from "@/lib/utils";

interface SidebarNavProps {
  /** Called after a link is chosen, so the mobile sheet can close itself. */
  onNavigate?: () => void;
  collapsed?: boolean;
  onToggleCollapse?: () => void;
}

const isActive = (pathname: string, href: string): boolean =>
  pathname === href || pathname.startsWith(`${href}/`);

export function SidebarNav({
  onNavigate,
  collapsed = false,
  onToggleCollapse,
}: SidebarNavProps) {
  const pathname = usePathname();
  const session = useSession();
  if (session.status !== "authenticated") return null;
  const { user, organization } = session.session;

  return (
    <div className="flex h-full min-h-0 flex-col">
      {/* Header / Logo */}
      <div
        className={cn(
          "flex h-14 shrink-0 items-center border-b border-sidebar-border px-4",
          collapsed ? "justify-center px-2" : "justify-between",
        )}
      >
        <Link
          href="/app/overview"
          onClick={onNavigate}
          className="rounded-md focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-sidebar-ring"
          title="Sentinel AI Gateway"
        >
          <Logo markOnly={collapsed} className="text-sidebar-foreground" />
        </Link>
      </div>

      {/* Organization Switcher */}
      <div
        className={cn(
          "shrink-0 border-b border-sidebar-border py-2.5",
          collapsed ? "px-2 text-center" : "px-3",
        )}
      >
        {collapsed ? (
          <div className="flex justify-center">
            <span
              className="flex size-8 items-center justify-center rounded-lg bg-white/10 font-mono text-xs font-bold text-sidebar-foreground"
              title={`Organization: ${organization.name}`}
            >
              {organization.name.slice(0, 2).toUpperCase()}
            </span>
          </div>
        ) : (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <button
                type="button"
                className="flex w-full items-center justify-between rounded-lg px-2.5 py-1.5 text-left transition-colors hover:bg-white/5 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-sidebar-ring"
              >
                <div className="min-w-0 pr-2">
                  <p className="text-[10px] font-semibold uppercase tracking-wider text-sidebar-muted">
                    Workspace
                  </p>
                  <p className="truncate text-sm font-semibold text-sidebar-foreground">
                    {organization.name}
                  </p>
                </div>
                <ChevronDown className="size-4 shrink-0 text-sidebar-muted" aria-hidden />
              </button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="start" className="w-56 glass">
              <DropdownMenuLabel className="text-xs">Organizations</DropdownMenuLabel>
              <DropdownMenuItem className="text-xs font-medium">
                <Building2 className="mr-2 size-3.5" aria-hidden />
                {organization.name}
                <span className="ml-auto text-[10px] text-emerald-500 font-semibold">Active</span>
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem asChild className="text-xs">
                <Link href="/app/organization/settings">
                  <Plus className="mr-2 size-3.5" aria-hidden /> Manage Workspaces
                </Link>
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        )}
      </div>

      {/* Navigation Tree */}
      <nav className="min-h-0 flex-1 overflow-y-auto sidebar-scrollbar px-2.5 py-4">
        <ul className="space-y-4">
          {DASHBOARD_NAV.map((section) => {
            const Icon = section.icon;

            if (section.children.length === 0) {
              const active = isActive(pathname, section.href);
              return (
                <li key={section.title}>
                  <Link
                    href={section.href}
                    onClick={onNavigate}
                    title={collapsed ? section.title : undefined}
                    aria-current={active ? "page" : undefined}
                    className={cn(
                      "flex items-center rounded-xl transition-all duration-200",
                      collapsed ? "justify-center size-10 mx-auto" : "gap-3 px-3 py-2 text-sm font-medium",
                      active
                        ? "bg-sidebar-active text-sidebar-accent-foreground shadow-xs border-l-2 border-primary"
                        : "text-sidebar-muted hover:bg-white/5 hover:text-sidebar-foreground",
                    )}
                  >
                    <Icon className="size-4 shrink-0" aria-hidden />
                    {!collapsed && <span>{section.title}</span>}
                  </Link>
                </li>
              );
            }

            return (
              <li key={section.title}>
                {!collapsed ? (
                  <>
                    <div className="flex items-center gap-2 px-3 pb-1 text-[10px] font-bold uppercase tracking-wider text-sidebar-muted">
                      <Icon className="size-3.5 shrink-0 opacity-70" aria-hidden />
                      {section.title}
                    </div>
                    <ul className="space-y-0.5 border-l border-sidebar-border/50 ml-4 pl-2.5">
                      {section.children.map((child) => {
                        const active = pathname === child.href;
                        return (
                          <li key={child.href}>
                            <Link
                              href={child.href}
                              onClick={onNavigate}
                              aria-current={active ? "page" : undefined}
                              className={cn(
                                "block truncate rounded-lg px-2.5 py-1.5 text-xs font-medium transition-colors",
                                active
                                  ? "bg-sidebar-active text-sidebar-accent-foreground font-semibold shadow-xs"
                                  : "text-sidebar-muted hover:bg-white/5 hover:text-sidebar-foreground",
                              )}
                            >
                              {child.label}
                            </Link>
                          </li>
                        );
                      })}
                    </ul>
                  </>
                ) : (
                  <div className="space-y-1">
                    {section.children.map((child) => {
                      const active = pathname === child.href;
                      return (
                        <Link
                          key={child.href}
                          href={child.href}
                          onClick={onNavigate}
                          title={`${section.title}: ${child.label}`}
                          aria-current={active ? "page" : undefined}
                          className={cn(
                            "flex items-center justify-center size-9 mx-auto rounded-lg transition-colors",
                            active
                              ? "bg-sidebar-active text-sidebar-accent-foreground shadow-xs"
                              : "text-sidebar-muted hover:bg-white/5 hover:text-sidebar-foreground",
                          )}
                        >
                          <Icon className="size-4 shrink-0" aria-hidden />
                        </Link>
                      );
                    })}
                  </div>
                )}
              </li>
            );
          })}
        </ul>
      </nav>

      {/* Footer / Utilities & Collapse Toggle */}
      <div className="shrink-0 space-y-1 border-t border-sidebar-border px-2.5 py-3">
        <Link
          href="/app/organization/settings"
          onClick={onNavigate}
          title={collapsed ? "Settings" : undefined}
          className={cn(
            "flex items-center rounded-lg text-xs font-medium text-sidebar-muted transition-colors hover:bg-white/5 hover:text-sidebar-foreground",
            collapsed ? "justify-center size-9 mx-auto" : "gap-2.5 px-2.5 py-1.5",
          )}
        >
          <Settings className="size-4 shrink-0" aria-hidden />
          {!collapsed && <span>Settings</span>}
        </Link>

        <Link
          href="/app/developer/quick-start"
          onClick={onNavigate}
          title={collapsed ? "Help & Documentation" : undefined}
          className={cn(
            "flex items-center rounded-lg text-xs font-medium text-sidebar-muted transition-colors hover:bg-white/5 hover:text-sidebar-foreground",
            collapsed ? "justify-center size-9 mx-auto" : "gap-2.5 px-2.5 py-1.5",
          )}
        >
          <LifeBuoy className="size-4 shrink-0" aria-hidden />
          {!collapsed && <span>Docs &amp; Help</span>}
        </Link>

        {/* User preview */}
        <div
          className={cn(
            "mt-2 flex items-center rounded-lg",
            collapsed ? "justify-center p-1" : "gap-2.5 px-2.5 py-1.5",
          )}
        >
          <Avatar className="size-7 ring-1 ring-white/20">
            <AvatarFallback className="bg-primary/20 text-sidebar-foreground text-[10px] font-bold">
              {initials(user.name)}
            </AvatarFallback>
          </Avatar>
          {!collapsed && (
            <span className="min-w-0 flex-1 truncate text-xs font-medium text-sidebar-foreground">
              {user.name}
            </span>
          )}
        </div>

        {/* Desktop Collapse Toggle */}
        {onToggleCollapse && (
          <div className="pt-1 hidden lg:block">
            <button
              type="button"
              onClick={onToggleCollapse}
              className={cn(
                "flex w-full items-center justify-center rounded-lg py-1.5 text-xs text-sidebar-muted hover:bg-white/5 hover:text-sidebar-foreground transition-colors",
                collapsed ? "px-1" : "gap-2 px-2.5",
              )}
              title={collapsed ? "Expand sidebar" : "Collapse sidebar"}
              aria-label={collapsed ? "Expand sidebar" : "Collapse sidebar"}
            >
              {collapsed ? (
                <ChevronRight className="size-4" aria-hidden />
              ) : (
                <>
                  <ChevronLeft className="size-4" aria-hidden />
                  <span className="text-[11px]">Collapse</span>
                </>
              )}
            </button>
          </div>
        )}
      </div>
    </div>
  );
}
