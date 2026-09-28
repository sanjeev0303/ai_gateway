"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import type { ReactNode } from "react";


import { ApiError, canRead } from "@/lib/control-api";
import { GROUPS, SECTIONS } from "@/lib/section-registry";
import { site } from "@/lib/site";
import { useSession } from "@/lib/queries";
import { ThemeSwitch } from "@/components/theme-switch";
import { AppShell, Sidebar as UISidebar, Topbar, UserPresence } from "@/components/ui";

function isUnauthenticated(error: unknown): boolean {
  return error instanceof ApiError && (error.status === 401 || error.status === 403);
}

export default function DashboardLayout({ children }: { readonly children: ReactNode }) {
  const session = useSession();
  const pathname = usePathname();

  if (session.isError && isUnauthenticated(session.error)) {
    return (
      <AppShell
        sidebar={<div className="p-4">Sign in required</div>}
        topbar={<Topbar><div>{site.name}</div></Topbar>}
      >
        <div className="flex flex-col items-center justify-center p-12 text-center border rounded-lg bg-muted/20 border-dashed">
          <h1 className="text-2xl font-bold mb-4">Sign in required</h1>
          <p className="text-muted-foreground mb-6">
            Your session is no longer valid.
          </p>
          <Link href="/login" className="btn btn-primary">Sign in to continue</Link>
        </div>
      </AppShell>
    );
  }

  const role = session.data?.claims.role ?? "";
  const userName = session.data?.claims.subject ?? "User"; // Fallback to subject or "User"

  return (
    <AppShell
      sidebar={<Sidebar pathname={pathname} role={role} />}
      topbar={
        <Topbar>
          <div className="flex items-center gap-4">
            <span className="font-semibold">{site.name} Control Plane</span>
          </div>
          <div className="flex items-center gap-4 text-sm">
            <button className="btn btn-ghost" style={{ fontSize: '0.75rem' }}>
              Search (Cmd+K)
            </button>
            <ThemeSwitch />
            <div style={{ width: '1px', height: '1.5rem', background: 'var(--line)' }} />
            <UserPresence name={userName} online={true} />
            <Link href="/logout" className="text-muted">
              [LogOut]
            </Link>
          </div>
        </Topbar>
      }
    >
      {children}
    </AppShell>
  );
}

function Sidebar({ pathname, role }: { readonly pathname: string; readonly role: string }) {
  const visible = SECTIONS.filter((s) => canRead(role, s.requires));

  return (
    <UISidebar>
      <div className="sidebar-header">
        <Link href="/" style={{ color: 'var(--ink)' }}>
          {site.name}
        </Link>
      </div>
      
      <div className="sidebar-nav">
        <div className="sidebar-group">
          <Link
            href="/dashboard"
            className={`sidebar-item ${pathname === "/dashboard" ? "active" : ""}`}
          >
            Overview
          </Link>
        </div>

        {GROUPS.map((group) => {
          const items = visible.filter((s) => s.group === group);
          if (items.length === 0) {
            return null;
          }
          return (
            <div key={group} className="sidebar-group">
              <h3 className="sidebar-group-title">{group}</h3>
              <div>
                {items.map((section) => {
                  const href = `/dashboard/${section.slug}`;
                  const active = pathname === href || pathname.startsWith(`${href}/`);
                  return (
                    <Link
                      key={section.slug}
                      href={href}
                      className={`sidebar-item ${active ? "active" : ""}`}
                    >
                      {section.title}
                    </Link>
                  );
                })}
              </div>
            </div>
          );
        })}
      </div>

      <div className="sidebar-group" style={{ marginTop: 'auto' }}>
        <Link
          href="/dashboard/settings"
          className={`sidebar-item ${pathname === "/dashboard/settings" ? "active" : ""}`}
        >
          Settings
        </Link>
      </div>
        {role !== "" && (
          <div className="sidebar-group" style={{ padding: '0 1rem', fontSize: '0.75rem', color: 'var(--ink-tertiary)' }}>
            Role: <span style={{ fontWeight: 600, color: 'var(--ink)' }}>{role}</span>
          </div>
        )}
    </UISidebar>
  );
}
