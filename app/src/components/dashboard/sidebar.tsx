"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { ChevronDown, LifeBuoy, Settings } from "lucide-react";

import { Logo } from "@/components/logo";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { DASHBOARD_NAV } from "@/lib/constants";
import { initials } from "@/lib/format";
import { useSession } from "@/lib/session-context";
import { cn } from "@/lib/utils";

interface SidebarNavProps {
  /** Called after a link is chosen, so the mobile sheet can close itself. */
  onNavigate?: () => void;
}

const isActive = (pathname: string, href: string): boolean =>
  pathname === href || pathname.startsWith(`${href}/`);

/**
 * The full navigation tree, reused by the desktop aside and the mobile sheet.
 * Childless sections render as a single link; sections with children render a
 * plain heading so the active state never lights up twice.
 */
export function SidebarNav({ onNavigate }: SidebarNavProps) {
  const pathname = usePathname();
  const session = useSession();
  if (session.status !== "authenticated") return null;
  const { user, organization } = session.session;

  return (
    <div className="flex h-full min-h-0 flex-col">
      <div className="flex h-14 shrink-0 items-center border-b border-sidebar-border px-4">
        <Link
          href="/app/overview"
          onClick={onNavigate}
          className="rounded-md focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-sidebar-ring"
        >
          <Logo className="text-sidebar-foreground" />
        </Link>
      </div>

      <div className="shrink-0 border-b border-sidebar-border px-3 py-3">
        <p className="px-2.5 text-[10px] font-semibold uppercase tracking-wider text-sidebar-muted">
          Organization
        </p>
        <p className="truncate px-2.5 pt-0.5 text-sm font-medium text-sidebar-foreground">
          {organization.name}
        </p>
      </div>

      <nav className="min-h-0 flex-1 overflow-y-auto px-3 py-4">
        <ul className="space-y-5">
          {DASHBOARD_NAV.map((section) => {
            const Icon = section.icon;

            if (section.children.length === 0) {
              const active = isActive(pathname, section.href);
              return (
                <li key={section.title}>
                  <Link
                    href={section.href}
                    onClick={onNavigate}
                    aria-current={active ? "page" : undefined}
                    className={cn(
                      "flex items-center gap-2.5 rounded-md px-2.5 py-2 text-sm font-medium transition-colors",
                      active
                        ? "bg-sidebar-active text-sidebar-accent-foreground"
                        : "text-sidebar-muted hover:bg-white/5 hover:text-sidebar-foreground",
                    )}
                  >
                    <Icon className="size-4 shrink-0" aria-hidden />
                    {section.title}
                  </Link>
                </li>
              );
            }

            return (
              <li key={section.title}>
                <div className="flex items-center gap-2.5 px-2.5 pb-1.5 text-[10px] font-semibold uppercase tracking-wider text-sidebar-muted">
                  <Icon className="size-3.5 shrink-0" aria-hidden />
                  {section.title}
                </div>
                <ul className="space-y-0.5 border-l border-sidebar-border pl-3">
                  {section.children.map((child) => {
                    const active = pathname === child.href;
                    return (
                      <li key={child.href}>
                        <Link
                          href={child.href}
                          onClick={onNavigate}
                          aria-current={active ? "page" : undefined}
                          className={cn(
                            "block truncate rounded-md px-2.5 py-1.5 text-[13px] transition-colors",
                            active
                              ? "bg-sidebar-active font-medium text-sidebar-accent-foreground"
                              : "text-sidebar-muted hover:bg-white/5 hover:text-sidebar-foreground",
                          )}
                        >
                          {child.label}
                        </Link>
                      </li>
                    );
                  })}
                </ul>
              </li>
            );
          })}
        </ul>
      </nav>

      <div className="shrink-0 space-y-1 border-t border-sidebar-border px-3 py-3">
        <Link
          href="/app/organization/settings"
          onClick={onNavigate}
          className="flex items-center gap-2.5 rounded-md px-2.5 py-1.5 text-[13px] text-sidebar-muted transition-colors hover:bg-white/5 hover:text-sidebar-foreground"
        >
          <Settings className="size-4 shrink-0" aria-hidden />
          Settings
        </Link>
        <Link
          href="/app/developer/quick-start"
          onClick={onNavigate}
          className="flex items-center gap-2.5 rounded-md px-2.5 py-1.5 text-[13px] text-sidebar-muted transition-colors hover:bg-white/5 hover:text-sidebar-foreground"
        >
          <LifeBuoy className="size-4 shrink-0" aria-hidden />
          Help
        </Link>
        <div className="mt-2 flex items-center gap-2.5 rounded-md px-2.5 py-1.5">
          <Avatar className="size-7">
            <AvatarFallback className="text-[11px]">
              {initials(user.name)}
            </AvatarFallback>
          </Avatar>
          <span className="min-w-0 flex-1 truncate text-[13px] text-sidebar-foreground">
            {user.name}
          </span>
          <ChevronDown className="size-3.5 shrink-0 text-sidebar-muted" aria-hidden />
        </div>
      </div>
    </div>
  );
}
