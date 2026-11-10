"use client";

import * as React from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  Activity,
  BellRing,
  Boxes,
  Building2,
  CreditCard,
  FileWarning,
  LayoutDashboard,
  LogOut,
  Menu,
  ScrollText,
  ServerCog,
  Users,
  type LucideIcon,
} from "lucide-react";

import { Logo } from "@/components/logo";
import { Button } from "@ai-governance/console-ui/ui/button";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetTitle,
  SheetTrigger,
} from "@ai-governance/console-ui/ui/sheet";
import { cn } from "@ai-governance/console-ui/lib/utils";
import { signOut } from "../../signin/actions";

interface NavItem {
  href: string;
  label: string;
  icon: LucideIcon;
}

const NAV: NavItem[] = [
  { href: "/", label: "Overview", icon: LayoutDashboard },
  { href: "/services", label: "Services", icon: ServerCog },
  { href: "/alerts", label: "Alerts", icon: BellRing },
  { href: "/logs", label: "Logs", icon: FileWarning },
  { href: "/billing", label: "Earnings", icon: CreditCard },
  { href: "/organizations", label: "Organizations", icon: Building2 },
  { href: "/users", label: "Users", icon: Users },
  { href: "/usage", label: "Usage", icon: Activity },
  { href: "/audit", label: "Audit", icon: ScrollText },
  { href: "/models", label: "Model catalog", icon: Boxes },
];

// Same row as the customer dashboard's sidebar, so both read as one product.
const ROW =
  "group relative flex h-9 w-full items-center gap-2.5 rounded-xl px-2.5 text-sm outline-none transition-[background-color,color,transform] duration-150 ease-(--ease-out-strong) focus-visible:ring-2 focus-visible:ring-sidebar-ring active:scale-[0.98]";
const ROW_IDLE = "text-sidebar-muted hover:bg-sidebar-accent hover:text-sidebar-foreground";
const ROW_ACTIVE =
  "bg-primary/10 font-medium text-primary before:absolute before:top-2 before:bottom-2 before:left-0 before:w-[3px] before:rounded-full before:bg-primary";

function isActive(pathname: string, href: string): boolean {
  if (href === "/") return pathname === "/";
  return pathname === href || pathname.startsWith(`${href}/`);
}

function AdminNav({ adminEmail, onNavigate }: { adminEmail: string; onNavigate?: () => void }) {
  const pathname = usePathname();

  return (
    <div className="flex h-full flex-col">
      <div className="flex flex-col gap-1 px-4 pt-4 pb-3">
        <Logo />
        <span className="text-xs text-sidebar-muted">Platform console</span>
      </div>

      <nav aria-label="Platform console" className="flex-1 overflow-y-auto px-2.5 py-2" data-lenis-prevent>
        <ul className="flex flex-col gap-0.5">
          {NAV.map((item) => {
            const active = isActive(pathname, item.href);
            const Icon = item.icon;
            return (
              <li key={item.href}>
                <Link
                  href={item.href}
                  onClick={onNavigate}
                  aria-current={active ? "page" : undefined}
                  className={cn(ROW, active ? ROW_ACTIVE : ROW_IDLE)}
                >
                  <Icon className="size-4 shrink-0" aria-hidden />
                  <span className="truncate">{item.label}</span>
                </Link>
              </li>
            );
          })}
        </ul>
      </nav>

      <div className="flex flex-col gap-2 border-t border-sidebar-border px-4 py-3">
        <span className="truncate text-xs text-sidebar-muted" title={adminEmail}>
          Signed in as {adminEmail}
        </span>
        <form action={signOut}>
          <button type="submit" className={cn(ROW, ROW_IDLE, "-mx-2.5 w-[calc(100%+1.25rem)]")}>
            <LogOut className="size-4 shrink-0" aria-hidden />
            Sign out
          </button>
        </form>
      </div>
    </div>
  );
}

function MobileAdminNav({ adminEmail }: { adminEmail: string }) {
  const [open, setOpen] = React.useState(false);
  return (
    <Sheet open={open} onOpenChange={setOpen}>
      <SheetTrigger asChild>
        <Button variant="ghost" size="icon" className="size-11 lg:hidden" aria-label="Open navigation">
          <Menu className="size-5" aria-hidden />
        </Button>
      </SheetTrigger>
      <SheetContent side="left" className="w-72 gap-0 bg-background p-0">
        <SheetTitle className="sr-only">Platform console navigation</SheetTitle>
        <SheetDescription className="sr-only">Pages of the platform console.</SheetDescription>
        <AdminNav adminEmail={adminEmail} onNavigate={() => setOpen(false)} />
      </SheetContent>
    </Sheet>
  );
}

/**
 * The platform console's frame: the customer dashboard's glass sidebar and
 * scroll area, with its own navigation. Nothing here belongs to one
 * organization.
 */
export function AdminShell({ adminEmail, children }: { adminEmail: string; children: React.ReactNode }) {
  return (
    <div className="relative flex h-screen overflow-hidden bg-background">
      <a
        href="#admin-main"
        className="sr-only focus:not-sr-only focus:absolute focus:top-3 focus:left-3 focus:z-50 focus:rounded-full focus:bg-primary focus:px-4 focus:py-2 focus:text-sm focus:text-primary-foreground"
      >
        Skip to content
      </a>
      <div aria-hidden className="ambient pointer-events-none absolute inset-x-0 top-0 h-[520px] opacity-80" />
      <aside className="relative hidden w-[17.5rem] shrink-0 py-3 pl-3 lg:block">
        <div className="liquid-glass-panel h-full overflow-hidden rounded-2xl border">
          <AdminNav adminEmail={adminEmail} />
        </div>
      </aside>

      <main id="admin-main" tabIndex={-1} className="relative min-h-0 flex-1 overflow-y-auto page-scrollbar outline-none">
        <div className="flex items-center gap-2 px-4 pt-3 lg:hidden">
          <MobileAdminNav adminEmail={adminEmail} />
          <span className="text-sm font-medium text-foreground">Platform console</span>
        </div>
        <div className="mx-auto w-full max-w-[1600px] px-4 py-6 sm:px-6 lg:px-8">{children}</div>
      </main>
    </div>
  );
}
