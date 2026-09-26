import Link from "next/link";
import type { ReactNode } from "react";

import { site } from "@/lib/site";

const navLinks = [
  { href: "/dashboard", label: "Overview" },
  { href: "/dashboard/settings", label: "Settings" },
] as const;

export default function DashboardLayout({ children }: { readonly children: ReactNode }) {
  return (
    <div className="dash">
      <aside className="dash-side">
        <Link href="/" className="brand">
          {site.name}
        </Link>
        <nav aria-label="Dashboard">
          {navLinks.map((link) => (
            <Link key={link.href} href={link.href}>
              {link.label}
            </Link>
          ))}
        </nav>
      </aside>
      <main className="dash-main">{children}</main>
    </div>
  );
}