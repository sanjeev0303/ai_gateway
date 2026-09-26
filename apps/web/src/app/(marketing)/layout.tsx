import Link from "next/link";
import type { ReactNode } from "react";

import { site } from "@/lib/site";

export default function MarketingLayout({ children }: { readonly children: ReactNode }) {
  return (
    <div>
      <header className="site-header container">
        <Link href="/" className="brand">
          {site.name}
        </Link>
        <nav aria-label="Main">
          <Link href="/login">Sign in</Link>
          <Link href="/signup" className="btn">
            Get started
          </Link>
        </nav>
      </header>
      <main className="container">{children}</main>
      <footer className="site-footer container">
        <p>© {new Date().getFullYear()} {site.name}</p>
      </footer>
    </div>
  );
}