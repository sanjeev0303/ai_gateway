"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import type { ReactNode } from "react";

import { ApiError, canRead } from "@/lib/control-api";
import { GROUPS, SECTIONS } from "@/lib/section-registry";
import { site } from "@/lib/site";
import { useSession } from "@/lib/queries";
import { ThemeSwitch } from "@/components/theme-switch";

function isUnauthenticated(error: unknown): boolean {
  return error instanceof ApiError && (error.status === 401 || error.status === 403);
}

export default function DashboardLayout({ children }: { readonly children: ReactNode }) {
  const session = useSession();
  const pathname = usePathname();

  if (session.isError && isUnauthenticated(session.error)) {
    return (
      <div className="dash">
        <Sidebar pathname={pathname} role="" />
        <main className="dash-main">
          <section className="dash-panel">
            <h1>Sign in required</h1>
            <p className="dash-note">
              Your session is no longer valid. <Link href="/login">Sign in</Link> to continue.
            </p>
          </section>
        </main>
      </div>
    );
  }

  return (
    <div className="dash">
      <Sidebar pathname={pathname} role={session.data?.claims.role ?? ""} />
      <main className="dash-main">{children}</main>
    </div>
  );
}

function Sidebar({ pathname, role }: { readonly pathname: string; readonly role: string }) {
  const visible = SECTIONS.filter((s) => canRead(role, s.requires));

  return (
    <aside className="dash-side">
      <Link href="/" className="brand">
        {site.name}
      </Link>
      <nav aria-label="Dashboard">
        <Link
          href="/dashboard"
          aria-current={pathname === "/dashboard" ? "page" : undefined}
          data-active={pathname === "/dashboard" || undefined}
        >
          Overview
        </Link>
        {GROUPS.map((group) => {
          const items = visible.filter((s) => s.group === group);
          if (items.length === 0) {
            return null;
          }
          return (
            <div key={group} className="nav-group">
              <p className="nav-group-label">{group}</p>
              {items.map((section) => {
                const href = `/dashboard/${section.slug}`;
                const active = pathname === href;
                return (
                  <Link
                    key={section.slug}
                    href={href}
                    aria-current={active ? "page" : undefined}
                    data-active={active || undefined}
                  >
                    {section.title}
                  </Link>
                );
              })}
            </div>
          );
        })}
        <div className="nav-group">
          <p className="nav-group-label">Settings</p>
          <Link
            href="/dashboard/settings"
            aria-current={pathname === "/dashboard/settings" ? "page" : undefined}
            data-active={pathname === "/dashboard/settings" || undefined}
          >
            Settings
          </Link>
        </div>
      </nav>
      <div className="dash-side-foot">
        <ThemeSwitch />
        {role !== "" && <p className="dash-note">Signed in as {role}</p>}
      </div>
    </aside>
  );
}
