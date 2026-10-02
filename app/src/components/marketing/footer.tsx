import Link from "next/link";

import { Logo } from "@/components/logo";
import {
  APP_DESCRIPTION,
  APP_NAME,
  APP_TAGLINE,
} from "@/lib/constants";

/** Every footer link points at a route this project actually serves. */
const COLUMNS = [
  {
    heading: "Product",
    links: [
      { label: "Overview", href: "/product" },
      { label: "Pricing", href: "/pricing" },
      { label: "Solutions", href: "/solutions" },
    ],
  },
  {
    heading: "Company",
    links: [
      { label: "About", href: "/company" },
      { label: "Resources", href: "/resources" },
    ],
  },
  {
    heading: "Dashboard",
    links: [
      { label: "Sign in", href: "/app/overview" },
      { label: "Live activity", href: "/app/observe/live-activity" },
      { label: "Policies", href: "/app/govern/policies" },
    ],
  },
] as const;

export function MarketingFooter() {
  return (
    <footer className="border-t bg-muted/30">
      <div className="mx-auto max-w-7xl px-4 py-12 sm:px-6">
        <div className="grid gap-10 md:grid-cols-[2fr_1fr_1fr_1fr]">
          <div>
            <Logo />
            <p className="mt-3 max-w-xs text-sm text-muted-foreground">
              {APP_DESCRIPTION}
            </p>
            <p className="mt-4 text-sm font-medium">{APP_TAGLINE}</p>
          </div>

          {COLUMNS.map((column) => (
            <div key={column.heading}>
              <h2 className="text-sm font-semibold">{column.heading}</h2>
              <ul className="mt-3 space-y-2">
                {column.links.map((link) => (
                  <li key={link.href}>
                    <Link
                      href={link.href}
                      className="text-sm text-muted-foreground transition-colors hover:text-foreground"
                    >
                      {link.label}
                    </Link>
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>

        <div className="mt-10 flex flex-col gap-2 border-t pt-6 text-sm text-muted-foreground sm:flex-row sm:items-center sm:justify-between">
          <p>
            © 2026 {APP_NAME}. All rights reserved.
          </p>
          <p>
            Built for teams that need to explain every AI request.
          </p>
        </div>
      </div>
    </footer>
  );
}
