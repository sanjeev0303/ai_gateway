import Link from "next/link";
import type { ReactNode } from "react";

import { BrandMark } from "@/components/icons";
import { SessionNav } from "@/components/session-nav";
import { site } from "@/lib/site";

import "./marketing.css";

const navLinks = [
  { href: "/#product", label: "Product" },
  { href: "/#pricing", label: "Pricing" },
  { href: "/docs", label: "Docs" },
] as const;

export default function MarketingLayout({ children }: { readonly children: ReactNode }) {
  return (
    <div className="landing">
      <header className="nav">
        <div className="l-wrap nav-inner">
          <Link href="/" className="nav-brand">
            <span className="nav-mark">
              <BrandMark width={18} height={18} />
            </span>
            {site.name}
          </Link>
          <nav aria-label="Main" className="nav-links">
            {navLinks.map((link) => (
              <Link key={link.href} href={link.href} className="nav-link">
                {link.label}
              </Link>
            ))}
          </nav>
          <SessionNav links={navLinks} />
        </div>
      </header>
      <main>{children}</main>
      <footer className="site-footer">
        <div className="l-wrap">
          <div className="footer-grid">
            <div>
              <span className="footer-brand">
                <span className="nav-mark">
                  <BrandMark width={18} height={18} />
                </span>
                {site.name}
              </span>
              <p className="footer-mission">
                One gateway for governed AI access. Scoped credentials,
                compiled policies, and budgets enforced before each request is
                forwarded.
              </p>
            </div>
            <div className="footer-col">
              <h3 className="footer-col-h">Product</h3>
              <ul>
                <li>
                  <Link href="/#product">Features</Link>
                </li>
                <li>
                  <Link href="/#architecture">Architecture</Link>
                </li>
                <li>
                  <Link href="/#pricing">Pricing</Link>
                </li>
              </ul>
            </div>
            <div className="footer-col">
              <h3 className="footer-col-h">Docs</h3>
              <ul>
                <li>
                  <Link href="/docs">Overview</Link>
                </li>
                <li>
                  <Link href="/docs/getting-started">Getting started</Link>
                </li>
                <li>
                  <Link href="/docs/api">API reference</Link>
                </li>
                <li>
                  <Link href="/docs/deployment">Deployment</Link>
                </li>
              </ul>
            </div>
            <div className="footer-col">
              <h3 className="footer-col-h">Organization</h3>
              <ul>
                <li>
                  <Link href="/signup">Get started</Link>
                </li>
                <li>
                  <Link href="/login">Sign in</Link>
                </li>
                <li>
                  <Link href="/docs/architecture">Architecture docs</Link>
                </li>
              </ul>
            </div>
          </div>
          <div className="footer-legal">
            <span>© {new Date().getFullYear()} {site.name}</span>
            <span>Self-hostable. Open source on GitHub.</span>
          </div>
        </div>
      </footer>
    </div>
  );
}