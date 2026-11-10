"use client";

import * as React from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  Building2,
  ChevronDown,
  ChevronsUpDown,
  ExternalLink,
  LogOut,
  PanelLeftClose,
  PanelLeftOpen,
  Settings,
  Users,
} from "lucide-react";

import { signOut } from "@/app/(auth)/signin/actions";
import { Logo } from "@/components/logo";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@ai-governance/console-ui/ui/dropdown-menu";
import { DASHBOARD_NAV, type NavSection } from "@/lib/constants";
import { initials } from "@ai-governance/console-ui/lib/format";
import { useSession } from "@/lib/session-context";
import { cn } from "@ai-governance/console-ui/lib/utils";

interface SidebarNavProps {
  /** Called after a link is chosen, so the mobile sheet can close itself. */
  onNavigate?: () => void;
  collapsed?: boolean;
  onToggleCollapse?: () => void;
}

const isActive = (pathname: string, href: string): boolean =>
  pathname === href || pathname.startsWith(`${href}/`);

const childActive = (pathname: string, child: NavSection["children"][number]): boolean =>
  (child.match ?? [child.href]).some((href) => isActive(pathname, href));

/** Three groups, so the list reads as watch, control, run. */
const NAV_GROUPS: { title: string; sectionTitles: string[] }[] = [
  { title: "Monitor", sectionTitles: ["Overview", "Observe"] },
  { title: "Control", sectionTitles: ["Govern", "Security", "Platform"] },
  { title: "Workspace", sectionTitles: ["Organization", "Billing", "Developer"] },
];

/* One row style for every link and section header, so the eye lands in one place. */
const ROW =
  "group relative flex h-9 w-full items-center gap-2.5 rounded-xl px-2.5 text-sm outline-none transition-[background-color,color,transform] duration-150 ease-(--ease-out-strong) focus-visible:ring-2 focus-visible:ring-sidebar-ring active:scale-[0.98]";
const ROW_IDLE = "text-sidebar-muted hover:bg-sidebar-accent hover:text-sidebar-foreground";
/* The page you are on: a green pill with an indicator at its left edge. */
const ROW_ACTIVE =
  "bg-primary/10 font-medium text-primary before:absolute before:top-2 before:bottom-2 before:left-0 before:w-[3px] before:rounded-full before:bg-primary";

export function SidebarNav({ onNavigate, collapsed = false, onToggleCollapse }: SidebarNavProps) {
  const pathname = usePathname();
  const session = useSession();

  // Sections start open when they hold the current page.
  const [openSections, setOpenSections] = React.useState<Record<string, boolean>>(() => {
    const initial: Record<string, boolean> = {};
    for (const section of DASHBOARD_NAV) {
      if (section.children.length > 0) {
        initial[section.title] = section.children.some((child) => childActive(pathname, child));
      }
    }
    return initial;
  });

  // Moving to a page inside a closed section opens it (adjusted during render, not in an effect).
  const [seenPath, setSeenPath] = React.useState(pathname);
  if (seenPath !== pathname) {
    setSeenPath(pathname);
    const section = DASHBOARD_NAV.find((s) => s.children.some((child) => childActive(pathname, child)));
    if (section && !openSections[section.title]) {
      setOpenSections((prev) => ({ ...prev, [section.title]: true }));
    }
  }

  const toggleSection = (title: string) => setOpenSections((prev) => ({ ...prev, [title]: !prev[title] }));

  const sectionMap = React.useMemo(() => new Map(DASHBOARD_NAV.map((sec) => [sec.title, sec])), []);

  if (session.status !== "authenticated") return null;
  const { user, organization } = session.session;
  const orgInitials = organization.name.slice(0, 2).toUpperCase();

  return (
    <div className="flex h-full min-h-0 flex-col select-none">
      {/* Brand and the collapse control, together where the eye starts */}
      <div className={cn("flex h-16 shrink-0 items-center gap-2", collapsed ? "flex-col justify-center pt-3 pb-1" : "justify-between px-4")}>
        <Link
          href="/app/overview"
          onClick={onNavigate}
          className="rounded-lg p-1 outline-none transition-transform focus-visible:ring-2 focus-visible:ring-sidebar-ring active:scale-95"
          aria-label="Tollseal overview"
        >
          <Logo markOnly={collapsed} className="text-sidebar-foreground" />
        </Link>
        {onToggleCollapse && !collapsed && (
          <button
            type="button"
            onClick={onToggleCollapse}
            className="hidden size-8 items-center justify-center rounded-lg text-sidebar-muted outline-none transition-[background-color,color] duration-150 hover:bg-sidebar-accent hover:text-sidebar-foreground focus-visible:ring-2 focus-visible:ring-sidebar-ring lg:flex"
            title="Collapse sidebar ( [ )"
            aria-label="Collapse sidebar"
          >
            <PanelLeftClose className="size-4" aria-hidden />
          </button>
        )}
      </div>
      {onToggleCollapse && collapsed && (
        <button
          type="button"
          onClick={onToggleCollapse}
          className="mx-auto mb-2 hidden size-9 items-center justify-center rounded-lg text-sidebar-muted outline-none transition-[background-color,color] duration-150 hover:bg-sidebar-accent hover:text-sidebar-foreground focus-visible:ring-2 focus-visible:ring-sidebar-ring lg:flex"
          title="Expand sidebar ( [ )"
          aria-label="Expand sidebar"
        >
          <PanelLeftOpen className="size-4" aria-hidden />
        </button>
      )}

      {/* Workspace */}
      <div className={cn("shrink-0 pb-3", collapsed ? "px-2" : "px-3")}>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <button
              type="button"
              title={collapsed ? `Workspace: ${organization.name}` : undefined}
              className={cn(
                "flex w-full items-center gap-2.5 rounded-xl border border-sidebar-border bg-card/70 text-left outline-none transition-[background-color,border-color,transform] duration-150 hover:border-border-strong hover:bg-card focus-visible:ring-2 focus-visible:ring-sidebar-ring active:scale-[0.99]",
                collapsed ? "mx-auto size-10 justify-center p-0" : "px-2.5 py-2",
              )}
            >
              <span className="flex size-7 shrink-0 items-center justify-center rounded-lg bg-primary text-[0.6875rem] font-semibold text-primary-foreground">
                {orgInitials}
              </span>
              {!collapsed && (
                <>
                  <span className="min-w-0 flex-1">
                    <span className="block truncate text-sm font-semibold text-sidebar-foreground">{organization.name}</span>
                    <span className="block text-xs text-sidebar-muted">Workspace</span>
                  </span>
                  <ChevronsUpDown className="size-3.5 shrink-0 text-sidebar-muted" aria-hidden />
                </>
              )}
            </button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="start" side={collapsed ? "right" : "bottom"} className="w-60">
            <DropdownMenuLabel className="text-xs text-muted-foreground">Workspace</DropdownMenuLabel>
            <DropdownMenuItem className="text-sm font-medium">
              <Building2 className="mr-2 size-4 text-primary" aria-hidden />
              <span className="truncate">{organization.name}</span>
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem asChild className="text-sm">
              <Link href="/app/organization/settings" onClick={onNavigate}>
                <Settings className="mr-2 size-4" aria-hidden /> Workspace settings
              </Link>
            </DropdownMenuItem>
            <DropdownMenuItem asChild className="text-sm">
              <Link href="/app/organization/members" onClick={onNavigate}>
                <Users className="mr-2 size-4" aria-hidden /> Members
              </Link>
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>

      <nav aria-label="Main" className="sidebar-scrollbar flex min-h-0 flex-1 flex-col gap-5 overflow-y-auto px-3 py-1">
        {NAV_GROUPS.map((group, groupIdx) => {
          const sections = group.sectionTitles
            .map((t) => sectionMap.get(t))
            .filter((s): s is NavSection => Boolean(s));

          return (
            <div key={group.title} className="flex flex-col gap-1">
              {collapsed ? (
                groupIdx > 0 && <div className="mx-auto mb-1 h-px w-6 bg-sidebar-border" aria-hidden />
              ) : (
                <p className="px-2.5 pb-0.5 text-xs font-medium text-sidebar-muted">{group.title}</p>
              )}

              <ul className="flex flex-col gap-0.5">
                {sections.map((section) => {
                  const Icon = section.icon;
                  const isLeaf = section.children.length === 0;
                  const active = isLeaf
                    ? isActive(pathname, section.href)
                    : section.children.some((c) => childActive(pathname, c));
                  const isOpen = Boolean(openSections[section.title]);

                  if (collapsed) {
                    if (isLeaf) {
                      return (
                        <li key={section.title}>
                          <Link
                            href={section.href}
                            onClick={onNavigate}
                            title={section.title}
                            aria-label={section.title}
                            aria-current={active ? "page" : undefined}
                            className={cn(ROW, "mx-auto size-10 justify-center px-0", active ? ROW_ACTIVE : ROW_IDLE)}
                          >
                            <Icon className="size-4 shrink-0" aria-hidden />
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
                              aria-label={section.title}
                              className={cn(ROW, "mx-auto size-10 justify-center px-0", active ? ROW_ACTIVE : ROW_IDLE)}
                            >
                              <Icon className="size-4 shrink-0" aria-hidden />
                            </button>
                          </DropdownMenuTrigger>
                          <DropdownMenuContent side="right" align="start" sideOffset={10} className="w-60">
                            <DropdownMenuLabel className="flex items-center gap-2 text-xs font-semibold text-foreground">
                              <Icon className="size-3.5 text-primary" aria-hidden />
                              {section.title}
                            </DropdownMenuLabel>
                            <DropdownMenuSeparator />
                            {section.children.map((child) => {
                              const childIsActive = childActive(pathname, child);
                              return (
                                <DropdownMenuItem key={child.href} asChild>
                                  <Link
                                    href={child.href}
                                    onClick={onNavigate}
                                    target={child.external ? "_blank" : undefined}
                                    rel={child.external ? "noopener" : undefined}
                                    aria-current={childIsActive ? "page" : undefined}
                                    className={cn("flex flex-col items-start gap-0 rounded-lg", childIsActive && "font-medium text-primary")}
                                  >
                                    <span className="text-sm">{child.label}</span>
                                    {child.description && (
                                      <span className="text-xs font-normal text-muted-foreground">{child.description}</span>
                                    )}
                                  </Link>
                                </DropdownMenuItem>
                              );
                            })}
                          </DropdownMenuContent>
                        </DropdownMenu>
                      </li>
                    );
                  }

                  if (isLeaf) {
                    return (
                      <li key={section.title}>
                        <Link
                          href={section.href}
                          onClick={onNavigate}
                          aria-current={active ? "page" : undefined}
                          className={cn(ROW, active ? ROW_ACTIVE : ROW_IDLE)}
                        >
                          <Icon className="size-4 shrink-0" aria-hidden />
                          <span className="truncate">{section.title}</span>
                        </Link>
                      </li>
                    );
                  }

                  const panelId = `nav-${section.title.toLowerCase().replace(/\s+/g, "-")}`;
                  return (
                    <li key={section.title}>
                      <button
                        type="button"
                        onClick={() => toggleSection(section.title)}
                        aria-expanded={isOpen}
                        aria-controls={panelId}
                        className={cn(ROW, active ? "font-medium text-sidebar-foreground hover:bg-sidebar-accent" : ROW_IDLE)}
                      >
                        <Icon className={cn("size-4 shrink-0", active && "text-primary")} aria-hidden />
                        <span className="flex-1 truncate text-left">{section.title}</span>
                        <ChevronDown
                          className={cn("size-3.5 shrink-0 text-sidebar-muted transition-transform duration-200 ease-(--ease-out-strong)", isOpen && "rotate-180")}
                          aria-hidden
                        />
                      </button>

                      {/* Opens and closes smoothly; closed items leave the tab order */}
                      <div
                        id={panelId}
                        inert={!isOpen}
                        className={cn(
                          "grid transition-[grid-template-rows,opacity] duration-200 ease-(--ease-out-strong) motion-reduce:transition-none",
                          isOpen ? "grid-rows-[1fr] opacity-100" : "grid-rows-[0fr] opacity-0",
                        )}
                      >
                        <ul className="ml-[1.0625rem] flex min-h-0 flex-col gap-0.5 overflow-hidden border-l border-sidebar-border pl-2.5">
                          {section.children.map((child) => {
                            const childIsActive = childActive(pathname, child);
                            return (
                              <li key={child.href} className="first:pt-1 last:pb-1">
                                <Link
                                  href={child.href}
                                  onClick={onNavigate}
                                  title={child.description}
                                  target={child.external ? "_blank" : undefined}
                                  rel={child.external ? "noopener" : undefined}
                                  aria-current={childIsActive ? "page" : undefined}
                                  className={cn(
                                    "flex h-8 items-center gap-2 rounded-lg px-2.5 text-[0.8125rem] outline-none transition-[background-color,color] duration-150 focus-visible:ring-2 focus-visible:ring-sidebar-ring",
                                    childIsActive
                                      ? "bg-primary/10 font-medium text-primary"
                                      : "text-sidebar-muted hover:bg-sidebar-accent hover:text-sidebar-foreground",
                                  )}
                                >
                                  <span className="truncate">{child.label}</span>
                                  {child.external && (
                                    <>
                                      <ExternalLink className="ml-auto size-3.5 shrink-0 opacity-70" aria-hidden />
                                      <span className="sr-only">(opens in a new tab)</span>
                                    </>
                                  )}
                                  {child.label === "Live activity" && (
                                    <span className="ml-auto size-1.5 shrink-0 rounded-full bg-emerald-500" aria-hidden />
                                  )}
                                </Link>
                              </li>
                            );
                          })}
                        </ul>
                      </div>
                    </li>
                  );
                })}
              </ul>
            </div>
          );
        })}
      </nav>

      {/* The account: settings and sign out */}
      <div className={cn("flex shrink-0 flex-col gap-1 border-t border-sidebar-border py-2.5", collapsed ? "px-2" : "px-3")}>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <button
              type="button"
              title={collapsed ? user.name : undefined}
              aria-label={`Account: ${user.name}`}
              className={cn(
                "flex w-full items-center gap-2.5 rounded-xl text-left outline-none transition-[background-color,transform] duration-150 hover:bg-sidebar-accent focus-visible:ring-2 focus-visible:ring-sidebar-ring active:scale-[0.99]",
                collapsed ? "mx-auto size-10 justify-center" : "px-2 py-1.5",
              )}
            >
              <span className="relative shrink-0">
                <Avatar className="size-8 ring-1 ring-sidebar-border">
                  <AvatarFallback className="bg-accent text-xs font-semibold text-primary">{initials(user.name)}</AvatarFallback>
                </Avatar>
                <span className="absolute -right-0.5 -bottom-0.5 size-2 rounded-full bg-emerald-500 ring-2 ring-card" aria-hidden />
              </span>
              {!collapsed && (
                <>
                  <span className="min-w-0 flex-1">
                    <span className="block truncate text-sm font-medium text-sidebar-foreground">{user.name}</span>
                    <span className="block truncate text-xs text-sidebar-muted">{user.email || "Signed in"}</span>
                  </span>
                  <ChevronsUpDown className="size-3.5 shrink-0 text-sidebar-muted" aria-hidden />
                </>
              )}
            </button>
          </DropdownMenuTrigger>
          <DropdownMenuContent side={collapsed ? "right" : "top"} align="start" sideOffset={8} className="w-60">
            <DropdownMenuLabel className="flex flex-col gap-0.5">
              <span className="truncate text-sm font-medium text-foreground">{user.name}</span>
              <span className="truncate text-xs font-normal text-muted-foreground">{user.email}</span>
            </DropdownMenuLabel>
            <DropdownMenuSeparator />
            <DropdownMenuItem asChild className="text-sm">
              <Link href="/app/organization/settings" onClick={onNavigate}>
                <Settings className="mr-2 size-4" aria-hidden /> Workspace settings
              </Link>
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <form action={signOut}>
              <DropdownMenuItem asChild variant="destructive" className="w-full text-sm">
                <button type="submit">
                  <LogOut className="mr-2 size-4" aria-hidden /> Sign out
                </button>
              </DropdownMenuItem>
            </form>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </div>
  );
}
