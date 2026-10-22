"use client";

import Link from "next/link";
import { Logo } from "@/components/logo";
import { APP_DESCRIPTION, APP_NAME, APP_TAGLINE } from "@/lib/constants";

const COLUMNS = [
  {
    heading: "Product",
    links: [
      { label: "Overview", href: "/product" },
      { label: "AI Gateway", href: "/product#gateway" },
      { label: "Policies", href: "/product#policies" },
      { label: "Audit Logs", href: "/product#audit" },
      { label: "Pricing", href: "/pricing" },
    ],
  },
  {
    heading: "Solutions",
    links: [
      { label: "Engineering", href: "/solutions#engineering" },
      { label: "Finance", href: "/solutions#finance" },
      { label: "Security", href: "/solutions#security" },
      { label: "Enterprise", href: "/solutions" },
    ],
  },
  {
    heading: "Developers",
    links: [
      { label: "Documentation", href: "/resources" },
      { label: "REST API", href: "/resources#api" },
      { label: "SDKs", href: "/resources#sdks" },
      { label: "Changelog", href: "/resources#changelog" },
    ],
  },
  {
    heading: "Company",
    links: [
      { label: "About", href: "/company" },
      { label: "Resources", href: "/resources" },
      { label: "Sign in", href: "/app/overview" },
    ],
  },
] as const;

export function MarketingFooter() {
  return (
    <footer
      className="border-t border-[rgba(229,228,226,0.08)] grain"
      style={{ background: "#1C1412" }}
    >
      <div className="mx-auto max-w-7xl px-6 sm:px-8 pt-16 pb-10">

        {/* Top: brand + columns */}
        <div className="grid gap-12 lg:grid-cols-[2fr_1fr_1fr_1fr_1fr]">

          {/* Brand column */}
          <div>
            <Logo className="text-[#E5E4E2]" />
            <p
              className="mt-5 text-sm leading-relaxed max-w-[30ch]"
              style={{ color: "rgba(229,228,226,0.50)", letterSpacing: "-0.01em" }}
            >
              {APP_DESCRIPTION}
            </p>
            <p
              className="mt-3 text-xs font-medium"
              style={{ color: "rgba(229,228,226,0.35)", fontFamily: "var(--font-fira-code)" }}
            >
              {APP_TAGLINE}
            </p>

            {/* Status badge */}
            <div
              className="mt-8 inline-flex items-center gap-2 border px-3 py-1.5"
              style={{
                borderColor: "rgba(74,122,90,0.25)",
                background: "rgba(74,122,90,0.06)",
              }}
            >
              <span
                className="size-1.5 rounded-full"
                style={{ background: "#5C8A6E" }}
                aria-hidden
              />
              <span
                className="text-[10px] font-medium"
                style={{ color: "#5C8A6E", fontFamily: "var(--font-fira-code)" }}
              >
                All Systems Operational
              </span>
            </div>
          </div>

          {/* Nav columns */}
          {COLUMNS.map((column) => (
            <div key={column.heading}>
              <h2
                className="text-[10px] font-semibold uppercase mb-5"
                style={{
                  fontFamily: "var(--font-fira-code)",
                  letterSpacing: "0.12em",
                  color: "rgba(229,228,226,0.35)",
                }}
              >
                {column.heading}
              </h2>
              <ul className="space-y-3">
                {column.links.map((link) => (
                  <li key={link.href}>
                    <Link
                      href={link.href}
                      className="text-sm transition-colors duration-150"
                      style={{ color: "rgba(229,228,226,0.55)", letterSpacing: "-0.01em" }}
                      onMouseEnter={e => (e.currentTarget.style.color = "#E5E4E2")}
                      onMouseLeave={e => (e.currentTarget.style.color = "rgba(229,228,226,0.55)")}
                    >
                      {link.label}
                    </Link>
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>

        {/* Bottom rule + copyright */}
        <div
          className="mt-16 pt-8 border-t flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between"
          style={{ borderColor: "rgba(229,228,226,0.07)" }}
        >
          <p
            className="text-xs"
            style={{ color: "rgba(229,228,226,0.30)", fontFamily: "var(--font-fira-code)" }}
          >
            &copy; 2026 {APP_NAME}. Enterprise AI Governance Platform.
          </p>
          <p
            className="text-xs"
            style={{ color: "rgba(229,228,226,0.22)", fontFamily: "var(--font-fira-code)" }}
          >
            Engineered for high-throughput, low-latency enterprise AI.
          </p>
        </div>
      </div>
    </footer>
  );
}
