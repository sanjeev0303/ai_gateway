"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

const docsNav = [
  { href: "/docs", label: "Overview" },
  { href: "/docs/getting-started", label: "Getting started" },
  { href: "/docs/architecture", label: "Architecture" },
  { href: "/docs/governance", label: "Governance" },
  { href: "/docs/api", label: "Control plane API" },
  { href: "/docs/deployment", label: "Deployment" },
] as const;

export function DocsSidebar() {
  const pathname = usePathname();
  return (
    <nav className="docs-sidebar" aria-label="Documentation">
      <p className="docs-sidebar-h">Documentation</p>
      <ul className="docs-sidebar-list">
        {docsNav.map((item) => {
          const isActive = pathname === item.href;
          return (
            <li key={item.href}>
              <Link
                href={item.href}
                className={
                  isActive ? "docs-sidebar-link is-active" : "docs-sidebar-link"
                }
                aria-current={isActive ? "page" : undefined}
              >
                {item.label}
              </Link>
            </li>
          );
        })}
      </ul>
      <div className="docs-sidebar-further">
        <p className="docs-sidebar-h">Further reading</p>
        <ul className="docs-sidebar-list">
          <li>
            <Link href="/#product" className="docs-sidebar-link">
              Product features
            </Link>
          </li>
          <li>
            <Link href="/#pricing" className="docs-sidebar-link">
              Pricing
            </Link>
          </li>
        </ul>
      </div>
    </nav>
  );
}