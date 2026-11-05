import Link from "next/link";
import { Logo } from "@/components/logo";
import { APP_DESCRIPTION, APP_NAME } from "@/lib/constants";

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
    <footer className="border-t border-border bg-card/50">
      <div className="mx-auto w-full max-w-[1240px] px-5 pt-16 pb-10 sm:px-8">
        <div className="grid gap-12 lg:grid-cols-[2fr_1fr_1fr_1fr_1fr]">
          <div className="flex flex-col gap-4">
            <Logo />
            <p className="max-w-[34ch] text-sm leading-relaxed text-muted-foreground">{APP_DESCRIPTION}</p>
          </div>

          {COLUMNS.map((column) => (
            <nav key={column.heading} aria-label={column.heading} className="flex flex-col gap-4">
              <h2 className="font-sans text-sm font-semibold tracking-normal text-foreground">{column.heading}</h2>
              <ul className="flex flex-col gap-2.5">
                {column.links.map((link) => (
                  <li key={link.href}>
                    <Link
                      href={link.href}
                      className="text-sm text-muted-foreground transition-colors duration-150 hover:text-foreground"
                    >
                      {link.label}
                    </Link>
                  </li>
                ))}
              </ul>
            </nav>
          ))}
        </div>

        <p aria-hidden className="mt-16 font-display text-[clamp(4rem,14vw,11rem)] leading-[0.8] tracking-[-0.04em] text-foreground/60 select-none">
          {APP_NAME}
        </p>
        <div className="mt-6 flex flex-col gap-2 border-t border-border pt-6 text-xs text-muted-foreground sm:flex-row sm:items-center sm:justify-between">
          <p>&copy; 2026 {APP_NAME}. AI gateway and governance.</p>
          <p>Self-hosted. Your keys and your data stay on your infrastructure.</p>
        </div>
      </div>
    </footer>
  );
}
