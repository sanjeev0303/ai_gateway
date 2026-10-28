"use client";

import * as React from "react";
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
  Sparkles,
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
import { DASHBOARD_NAV, type NavSection } from "@/lib/constants";
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

/** Three primary macro-domains for cognitive chunking (Miller's Law / 7±2) */
const NAV_GROUPS: { title: string; sectionTitles: string[] }[] = [
  {
    title: "Telemetry & Metrics",
    sectionTitles: ["Overview", "Observe"],
  },
  {
    title: "Control & Platform",
    sectionTitles: ["Govern", "Security", "Platform"],
  },
  {
    title: "Workspace & Dev",
    sectionTitles: ["Organization", "Billing", "Developer"],
  },
];

export function SidebarNav({
  onNavigate,
  collapsed = false,
  onToggleCollapse,
}: SidebarNavProps) {
  const pathname = usePathname();
  const session = useSession();

  // Accordion state tracking open/closed sections (Progressive Disclosure)
  const [openSections, setOpenSections] = React.useState<Record<string, boolean>>(() => {
    const initial: Record<string, boolean> = {};
    for (const section of DASHBOARD_NAV) {
      if (section.children.length > 0) {
        const hasActive = section.children.some((child) => isActive(pathname, child.href));
        // Auto-expand if current route belongs to this section, or default open "Observe"
        initial[section.title] = hasActive || section.title === "Observe";
      }
    }
    return initial;
  });

  // Auto-expand when route changes (Recognition over Recall)
  React.useEffect(() => {
    for (const section of DASHBOARD_NAV) {
      if (section.children.some((child) => isActive(pathname, child.href))) {
        setOpenSections((prev) => (prev[section.title] ? prev : { ...prev, [section.title]: true }));
      }
    }
  }, [pathname]);

  const toggleSection = (title: string) => {
    setOpenSections((prev) => ({ ...prev, [title]: !prev[title] }));
  };

  if (session.status !== "authenticated") return null;
  const { user, organization } = session.session;

  const sectionMap = React.useMemo(() => {
    const map = new Map<string, NavSection>();
    for (const sec of DASHBOARD_NAV) {
      map.set(sec.title, sec);
    }
    return map;
  }, []);

  return (
    <div className="flex h-full min-h-0 flex-col bg-sidebar select-none">
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
          className="rounded-lg p-1 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-sidebar-ring transition-transform active:scale-95"
          title="Sentinel AI Gateway"
        >
          <Logo markOnly={collapsed} className="text-sidebar-foreground" />
        </Link>
      </div>

      {/* Workspace Switcher (Common Region & Jakob's Law) */}
      <div
        className={cn(
          "shrink-0 border-b border-sidebar-border py-2.5",
          collapsed ? "px-2 text-center" : "px-3",
        )}
      >
        {collapsed ? (
          <div className="flex justify-center">
            <span
              className="flex size-9 items-center justify-center rounded-xl bg-white/10 font-mono text-xs font-bold text-sidebar-foreground ring-1 ring-white/15 shadow-2xs"
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
                className="flex w-full items-center justify-between rounded-xl px-2.5 py-1.5 text-left transition-all duration-150 hover:bg-white/[0.06] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-sidebar-ring active:scale-[0.99] border border-transparent hover:border-sidebar-border/60"
              >
                <div className="min-w-0 pr-2">
                  <div className="flex items-center gap-1.5">
                    <p className="text-xs font-semibold uppercase tracking-wider text-sidebar-muted">
                      Workspace
                    </p>
                    <span className="size-1.5 rounded-full bg-emerald-500" />
                  </div>
                  <p className="truncate text-sm font-semibold text-sidebar-foreground">
                    {organization.name}
                  </p>
                </div>
                <ChevronDown className="size-3.5 shrink-0 text-sidebar-muted/80" aria-hidden />
              </button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="start" className="w-56 glass border border-sidebar-border shadow-xl">
              <DropdownMenuLabel className="text-xs text-muted-foreground">Active Workspace</DropdownMenuLabel>
              <DropdownMenuItem className="text-sm font-medium cursor-pointer">
                <Building2 className="mr-2 size-4 text-primary" aria-hidden />
                <span className="truncate">{organization.name}</span>
                <span className="ml-auto text-xs text-emerald-400 font-semibold px-2 py-0.5 rounded-full bg-emerald-500/10">Active</span>
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem asChild className="text-xs cursor-pointer">
                <Link href="/app/organization/settings">
                  <Plus className="mr-2 size-3.5" aria-hidden /> Manage Workspaces
                </Link>
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        )}
      </div>

      {/* Navigation Tree (Structured Hierarchies & Progressive Disclosure) */}
      <nav className="min-h-0 flex-1 overflow-y-auto sidebar-scrollbar px-2.5 py-3 space-y-4">
        {NAV_GROUPS.map((group, groupIdx) => {
          const sections = group.sectionTitles
            .map((t) => sectionMap.get(t))
            .filter((s): s is NavSection => Boolean(s));

          return (
            <div key={group.title} className="space-y-1">
              {/* Domain Group Label (Miller's Law Chunking) */}
              {!collapsed ? (
                <div className="flex items-center justify-between px-2.5 pt-2 pb-1.5 text-xs font-bold uppercase tracking-wider text-sidebar-muted/80">
                  <span>{group.title}</span>
                  {groupIdx > 0 && <span className="h-px flex-1 ml-2 bg-sidebar-border/30" />}
                </div>
              ) : (
                groupIdx > 0 && <div className="h-px w-6 mx-auto my-1.5 bg-sidebar-border/40" />
              )}

              {/* Sections in this group */}
              <ul className="space-y-1">
                {sections.map((section) => {
                  const Icon = section.icon;
                  const isLeaf = section.children.length === 0;
                  const isParentActive =
                    isLeaf
                      ? isActive(pathname, section.href)
                      : section.children.some((c) => isActive(pathname, c.href));
                  const isOpen = Boolean(openSections[section.title]);

                  // Collapsed View: Icon button with DropdownMenu flyout
                  if (collapsed) {
                    if (isLeaf) {
                      return (
                        <li key={section.title}>
                          <Link
                            href={section.href}
                            onClick={onNavigate}
                            title={section.title}
                            aria-current={isParentActive ? "page" : undefined}
                            className={cn(
                              "flex items-center justify-center size-10 mx-auto rounded-xl transition-all duration-150 relative",
                              isParentActive
                                ? "bg-white/[0.08] text-white shadow-xs border border-white/10"
                                : "text-sidebar-muted hover:bg-white/[0.04] hover:text-sidebar-foreground",
                              "active:scale-95",
                            )}
                          >
                            <Icon className="size-4 shrink-0" aria-hidden />
                            {isParentActive && (
                              <span className="absolute top-1.5 right-1.5 size-1.5 rounded-full bg-emerald-500 ring-2 ring-sidebar" />
                            )}
                          </Link>
                        </li>
                      );
                    }

                    return (
                      <li key={section.title}>
                        <DropdownMenu>
                          <DropdownMenuTrigger asChild>
                            <button
                              type="button"
                              title={section.title}
                              className={cn(
                                "flex items-center justify-center size-10 mx-auto rounded-xl transition-all duration-150 relative focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-sidebar-ring",
                                isParentActive
                                  ? "bg-white/[0.08] text-white shadow-xs border border-white/10"
                                  : "text-sidebar-muted hover:bg-white/[0.04] hover:text-sidebar-foreground",
                                "active:scale-95",
                              )}
                            >
                              <Icon className="size-4 shrink-0" aria-hidden />
                              {isParentActive && (
                                <span className="absolute top-1.5 right-1.5 size-1.5 rounded-full bg-emerald-500 ring-2 ring-sidebar" />
                              )}
                            </button>
                          </DropdownMenuTrigger>
                          <DropdownMenuContent
                            side="right"
                            align="start"
                            sideOffset={10}
                            className="w-56 glass border border-sidebar-border shadow-xl p-1.5"
                          >
                            <DropdownMenuLabel className="flex items-center gap-2 px-2 py-1.5 text-xs font-semibold text-sidebar-foreground">
                              <Icon className="size-3.5 text-primary" aria-hidden />
                              {section.title}
                            </DropdownMenuLabel>
                            <DropdownMenuSeparator />
                            <div className="space-y-0.5">
                              {section.children.map((child) => {
                                const active = pathname === child.href;
                                return (
                                  <DropdownMenuItem key={child.href} asChild>
                                    <Link
                                      href={child.href}
                                      onClick={onNavigate}
                                      className={cn(
                                        "flex flex-col py-1.5 px-2.5 rounded-lg text-sm cursor-pointer transition-colors",
                                        active
                                          ? "bg-white/[0.08] text-white font-semibold border-l-2 border-primary"
                                          : "text-sidebar-muted hover:text-sidebar-foreground hover:bg-white/5",
                                      )}
                                    >
                                      <div className="flex items-center justify-between w-full">
                                        <span>{child.label}</span>
                                        {child.label === "Live activity" && (
                                          <span className="size-1.5 rounded-full bg-emerald-500 animate-pulse" />
                                        )}
                                      </div>
                                      {child.description && (
                                        <span className="text-xs text-muted-foreground/80 font-normal">
                                          {child.description}
                                        </span>
                                      )}
                                    </Link>
                                  </DropdownMenuItem>
                                );
                              })}
                            </div>
                          </DropdownMenuContent>
                        </DropdownMenu>
                      </li>
                    );
                  }

                  // Expanded View: Leaf link or Collapsible Accordion Group
                  if (isLeaf) {
                    return (
                      <li key={section.title}>
                        <Link
                          href={section.href}
                          onClick={onNavigate}
                          aria-current={isParentActive ? "page" : undefined}
                          className={cn(
                            "flex items-center gap-2.5 rounded-xl px-2.5 py-2 text-sm font-semibold transition-all duration-150 h-9.5",
                            isParentActive
                              ? "bg-white/[0.08] text-white font-semibold shadow-xs border-l-2 border-primary pl-2"
                              : "text-sidebar-muted hover:bg-white/[0.04] hover:text-sidebar-foreground",
                            "active:scale-[0.98]",
                          )}
                        >
                          <Icon className={cn("size-4 shrink-0 transition-colors", isParentActive ? "text-primary" : "text-sidebar-muted")} aria-hidden />
                          <span className="truncate">{section.title}</span>
                        </Link>
                      </li>
                    );
                  }

                  return (
                    <li key={section.title} className="space-y-0.5">
                      {/* Accordion Header (Hick's Law & Fitts's Law) */}
                      <button
                        type="button"
                        onClick={() => toggleSection(section.title)}
                        aria-expanded={isOpen}
                        className={cn(
                          "flex w-full items-center justify-between rounded-xl px-2.5 py-2 text-sm font-semibold transition-all duration-150 h-9.5",
                          isParentActive
                            ? "text-sidebar-foreground font-bold"
                            : "text-sidebar-muted hover:bg-white/[0.04] hover:text-sidebar-foreground",
                          "active:scale-[0.99] focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-sidebar-ring",
                        )}
                      >
                        <div className="flex items-center gap-2.5 min-w-0">
                          <Icon className={cn("size-4 shrink-0 transition-colors", isParentActive ? "text-primary" : "text-sidebar-muted")} aria-hidden />
                          <span className="truncate">{section.title}</span>
                          {isParentActive && (
                            <span className="size-1.5 rounded-full bg-emerald-500 shrink-0" />
                          )}
                        </div>
                        <ChevronDown
                          className={cn(
                            "size-3.5 shrink-0 text-sidebar-muted/70 transition-transform duration-200",
                            isOpen && "rotate-180 text-sidebar-foreground",
                          )}
                          aria-hidden
                        />
                      </button>

                      {/* Accordion Subitems (Visual Hierarchy & Law of Proximity) */}
                      {isOpen && (
                        <ul className="space-y-0.5 border-l border-sidebar-border/50 ml-4 pl-2.5 py-0.5">
                          {section.children.map((child) => {
                            const active = pathname === child.href;
                            const isLive = child.label === "Live activity";
                            const isAlerts = child.label === "Alerts";
                            const isQuickStart = child.label === "Quick start";

                            return (
                              <li key={child.href}>
                                <Link
                                  href={child.href}
                                  onClick={onNavigate}
                                  aria-current={active ? "page" : undefined}
                                  className={cn(
                                    "flex items-center justify-between truncate rounded-lg px-2.5 py-1.5 text-sm font-medium transition-all duration-150 h-8.5",
                                    active
                                      ? "bg-white/[0.08] text-white font-semibold shadow-xs border-l-2 border-primary pl-2"
                                      : "text-sidebar-muted hover:bg-white/[0.04] hover:text-sidebar-foreground",
                                    "active:scale-[0.98]",
                                  )}
                                >
                                  <span className="truncate">{child.label}</span>
                                  {isLive && (
                                    <span className="flex items-center gap-1 text-xs text-emerald-400 font-mono">
                                      <span className="size-1.5 rounded-full bg-emerald-500 animate-pulse" />
                                      Live
                                    </span>
                                  )}
                                  {isAlerts && (
                                    <span className="text-xs px-2 py-0.5 rounded-full bg-amber-500/15 text-amber-400 border border-amber-500/20 font-medium">
                                      Watch
                                    </span>
                                  )}
                                  {isQuickStart && (
                                    <span className="text-xs px-2 py-0.5 rounded-full bg-primary/20 text-primary-foreground border border-primary/30 font-medium flex items-center gap-0.5">
                                      <Sparkles className="size-3" />
                                      Guide
                                    </span>
                                  )}
                                </Link>
                              </li>
                            );
                          })}
                        </ul>
                      )}
                    </li>
                  );
                })}
              </ul>
            </div>
          );
        })}
      </nav>

      {/* Footer / Utilities & Collapse Toggle (Serial Position Effect) */}
      <div className="shrink-0 space-y-1 border-t border-sidebar-border px-2.5 py-2.5 bg-sidebar/80">
        <Link
          href="/app/organization/settings"
          onClick={onNavigate}
          title={collapsed ? "Settings" : undefined}
          className={cn(
            "flex items-center rounded-xl text-sm font-medium text-sidebar-muted transition-all hover:bg-white/[0.04] hover:text-sidebar-foreground h-9.5",
            collapsed ? "justify-center size-10 mx-auto" : "gap-2.5 px-2.5",
            "active:scale-[0.98]",
          )}
        >
          <Settings className="size-4 shrink-0 text-sidebar-muted" aria-hidden />
          {!collapsed && <span>Settings</span>}
        </Link>

        <Link
          href="/app/developer/quick-start"
          onClick={onNavigate}
          title={collapsed ? "Help & Documentation" : undefined}
          className={cn(
            "flex items-center rounded-xl text-sm font-medium text-sidebar-muted transition-all hover:bg-white/[0.04] hover:text-sidebar-foreground h-9.5",
            collapsed ? "justify-center size-10 mx-auto" : "gap-2.5 px-2.5",
            "active:scale-[0.98]",
          )}
        >
          <LifeBuoy className="size-4 shrink-0 text-sidebar-muted" aria-hidden />
          {!collapsed && <span>Docs &amp; Help</span>}
        </Link>

        {/* User Identity Preview with Beacon */}
        <div
          className={cn(
            "mt-1.5 flex items-center rounded-xl bg-white/[0.03] border border-sidebar-border/40 p-1.5",
            collapsed ? "justify-center" : "gap-2.5 px-2.5 py-1.5",
          )}
        >
          <div className="relative shrink-0">
            <Avatar className="size-7 ring-1 ring-white/20">
              <AvatarFallback className="bg-primary/20 text-sidebar-foreground text-xs font-bold">
                {initials(user.name)}
              </AvatarFallback>
            </Avatar>
            <span className="absolute -bottom-0.5 -right-0.5 size-2 rounded-full bg-emerald-500 ring-2 ring-sidebar" />
          </div>
          {!collapsed && (
            <div className="min-w-0 flex-1">
              <p className="truncate text-sm font-medium text-sidebar-foreground">{user.name}</p>
              <p className="truncate text-xs text-sidebar-muted">{user.email || "Active User"}</p>
            </div>
          )}
        </div>

        {/* Desktop Collapse Toggle (Fitts's Law + Keyboard Shortcut Hint) */}
        {onToggleCollapse && (
          <div className="pt-1 hidden lg:block">
            <button
              type="button"
              onClick={onToggleCollapse}
              className={cn(
                "flex w-full items-center justify-between rounded-xl py-1.5 text-xs text-sidebar-muted hover:bg-white/[0.05] hover:text-sidebar-foreground transition-all h-9",
                collapsed ? "justify-center px-1" : "px-2.5",
                "active:scale-[0.98] border border-transparent hover:border-sidebar-border/50",
              )}
              title={collapsed ? "Expand sidebar ([)" : "Collapse sidebar ([)"}
              aria-label={collapsed ? "Expand sidebar" : "Collapse sidebar"}
            >
              {collapsed ? (
                <ChevronRight className="size-4" aria-hidden />
              ) : (
                <>
                  <div className="flex items-center gap-2">
                    <ChevronLeft className="size-4" aria-hidden />
                    <span className="text-xs font-medium">Collapse sidebar</span>
                  </div>
                  <kbd className="hidden font-mono text-xs uppercase tracking-wider px-2 py-0.5 rounded bg-white/5 border border-white/10 text-muted-foreground sm:inline-block">
                    [
                  </kbd>
                </>
              )}
            </button>
          </div>
        )}
      </div>
    </div>
  );
}
