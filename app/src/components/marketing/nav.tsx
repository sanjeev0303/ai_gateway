"use client";

import { useEffect, useState, useSyncExternalStore } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { LayoutDashboard, Moon, Sun, Monitor, Menu, X } from "lucide-react";
import { useTheme } from "next-themes";
import { Logo } from "@/components/logo";
import { MARKETING_NAV } from "@/lib/constants";
import { cn } from "@ai-governance/console-ui/lib/utils";
import { useLenis } from "@/components/providers/lenis-provider";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "@ai-governance/console-ui/ui/dropdown-menu";

const emptySubscribe = () => () => {};

function ThemeToggle() {
  const { theme, resolvedTheme, setTheme } = useTheme();
  const mounted = useSyncExternalStore(emptySubscribe, () => true, () => false);

  if (!mounted) return (
    <button
      className="flex size-9 items-center justify-center rounded-full text-muted-foreground"
      aria-label="Toggle theme"
    >
      <Sun className="size-4 opacity-40" aria-hidden />
    </button>
  );

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          className="flex size-9 items-center justify-center rounded-full text-muted-foreground transition-colors duration-150 hover:bg-accent hover:text-foreground focus-visible:ring-3 focus-visible:ring-ring/30 focus-visible:outline-none"
          aria-label="Toggle theme"
        >
          {resolvedTheme === "dark" ? <Moon className="size-4" aria-hidden /> : <Sun className="size-4" aria-hidden />}
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent
        align="end"
        className="w-36"
      >
        <DropdownMenuLabel className="text-xs font-medium text-muted-foreground">
          Theme
        </DropdownMenuLabel>
        <DropdownMenuItem onClick={() => setTheme("light")} className="text-sm">
          <Sun className="mr-2 size-3.5" aria-hidden /> Light
        </DropdownMenuItem>
        <DropdownMenuItem onClick={() => setTheme("dark")} className="text-sm">
          <Moon className="mr-2 size-3.5" aria-hidden /> Dark
        </DropdownMenuItem>
        <DropdownMenuItem onClick={() => setTheme("system")} className="text-sm">
          <Monitor className="mr-2 size-3.5" aria-hidden /> System
          {theme === "system" && <span className="ml-auto text-xs text-muted-foreground">active</span>}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

export function MarketingNav({ className, signedIn = false }: { className?: string; signedIn?: boolean }) {
  const pathname = usePathname();
  const [open, setOpen] = useState(false);
  const [scrolled, setScrolled] = useState(false);
  const [activeSection, setActiveSection] = useState<string>("");
  const { scrollTo } = useLenis();

  useEffect(() => {
    const handleScroll = () => setScrolled(window.scrollY > 20);
    handleScroll();
    window.addEventListener("scroll", handleScroll, { passive: true });
    return () => window.removeEventListener("scroll", handleScroll);
  }, []);

  useEffect(() => {
    if (pathname !== "/") return;
    const sectionIds = MARKETING_NAV.map((item) => item.href.replace("/#", "")).filter(Boolean);
    const observer = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (entry.isIntersecting) {
            setActiveSection(entry.target.id);
          }
        }
      },
      { rootMargin: "-15% 0px -65% 0px" }
    );

    sectionIds.forEach((id) => {
      const el = document.getElementById(id);
      if (el) observer.observe(el);
    });

    return () => observer.disconnect();
  }, [pathname]);

  // On the homepage a section link scrolls instead of navigating. Sections carry
  // their own scroll-margin for the header, so no extra offset is added here;
  // a link whose section is missing falls through to a normal navigation.
  const handleNavClick = (e: React.MouseEvent<HTMLAnchorElement>, href: string) => {
    if (!href.startsWith("/#") || pathname !== "/") return;
    const targetId = href.replace("/#", "");
    const element = document.getElementById(targetId);
    if (!element) return;
    e.preventDefault();
    scrollTo(element, { offset: 0 });
    window.history.pushState(null, "", `#${targetId}`);
    setActiveSection(targetId);
  };

  return (
    <>
      {/* One glass bar floating over the page: logo, links and actions share a single surface. */}
      <header className={cn("sticky top-0 z-50 px-3 pt-3 sm:px-5", className)}>
        <div
          className={cn(
            "liquid-glass mx-auto flex h-14 w-full max-w-[1240px] items-center gap-6 rounded-full border pr-2 pl-5 transition-[box-shadow] duration-300 sm:gap-8",
            scrolled && "shadow-(--shadow-float)",
          )}
        >
          {/* Logo */}
          <Link
            href="/"
            onClick={(e) => {
              if (pathname === "/") {
                e.preventDefault();
                scrollTo(0);
                window.history.pushState(null, "", "/");
              }
            }}
            className="shrink-0 rounded-lg focus-visible:ring-3 focus-visible:ring-ring/30 focus-visible:outline-none"
          >
            <Logo />
          </Link>

          {/* Desktop nav */}
          <nav aria-label="Main" className="ml-2 hidden md:block">
            <ul className="flex items-center gap-0">
              {MARKETING_NAV.map((item) => {
                const sectionId = item.href.replace("/#", "");
                const active = pathname === "/" ? activeSection === sectionId : pathname === item.href;
                return (
                  <li key={item.href}>
                    <Link
                      href={item.href}
                      onClick={(e) => handleNavClick(e, item.href)}
                      aria-current={active ? "page" : undefined}
                      className={cn(
                        "block rounded-full px-3.5 py-2 text-sm transition-colors duration-150 focus-visible:ring-3 focus-visible:ring-ring/30 focus-visible:outline-none",
                        active ? "font-medium text-primary" : "text-foreground/75 hover:text-foreground",
                      )}
                    >
                      {item.label}
                    </Link>
                  </li>
                );
              })}
            </ul>
          </nav>

          {/* Right: actions */}
          <div className="ml-auto flex items-center gap-3">
            <div className="hidden sm:block">
              <ThemeToggle />
            </div>

            {signedIn ? (
              <Link
                href="/app/overview"
                id="nav-dashboard"
                className="inline-flex h-9 items-center gap-1.5 rounded-full bg-primary px-4 text-sm font-medium whitespace-nowrap text-primary-foreground sm:px-5 transition-[background-color,transform] duration-150 hover:bg-primary-hover focus-visible:ring-3 focus-visible:ring-ring/30 focus-visible:outline-none active:scale-[0.97]"
              >
                <LayoutDashboard className="size-4" aria-hidden />
                Dashboard
              </Link>
            ) : (
              <>
                <Link
                  href="/signin"
                  id="nav-sign-in"
                  className="hidden h-9 items-center rounded-full px-4 text-sm font-medium text-foreground/80 transition-colors duration-150 hover:text-foreground focus-visible:ring-3 focus-visible:ring-ring/30 focus-visible:outline-none sm:inline-flex"
                >
                  Sign in
                </Link>
                <Link
                  href="/signup"
                  id="nav-get-started"
                  className="inline-flex h-9 items-center rounded-full bg-primary px-4 text-sm font-medium whitespace-nowrap text-primary-foreground sm:px-5 transition-[background-color,transform] duration-150 hover:bg-primary-hover focus-visible:ring-3 focus-visible:ring-ring/30 focus-visible:outline-none active:scale-[0.97]"
                >
                  Get started
                </Link>
              </>
            )}

            {/* Mobile menu toggle */}
            <button
              type="button"
              onClick={() => setOpen(!open)}
              className="flex size-9 items-center justify-center rounded-full border border-border-strong text-foreground md:hidden focus-visible:ring-3 focus-visible:ring-ring/30 focus-visible:outline-none"
              aria-label={open ? "Close menu" : "Open menu"}
              aria-expanded={open}
            >
              {open ? <X className="size-4" aria-hidden /> : <Menu className="size-4" aria-hidden />}
            </button>
          </div>
        </div>
      </header>

      {/* Mobile drawer */}
      {open && (
        <div
          className="fixed inset-0 z-40 bg-[rgba(20,26,51,0.32)] backdrop-blur-[2px] md:hidden"
          onClick={() => setOpen(false)}
          aria-hidden
        />
      )}
      <nav
        aria-label="Mobile"
        className={cn(
          "liquid-glass-panel fixed top-[4.75rem] right-3 bottom-3 z-50 w-72 overflow-y-auto rounded-2xl border transition-transform duration-300 ease-(--ease-drawer) md:hidden",
          open ? "visible translate-x-0" : "invisible translate-x-[calc(100%+1rem)]"
        )}
      >
        <ul className="flex flex-col p-3">
          {MARKETING_NAV.map((item) => {
            const sectionId = item.href.replace("/#", "");
            const active = pathname === "/" ? activeSection === sectionId : pathname === item.href;
            return (
              <li key={item.href}>
                <Link
                  href={item.href}
                  onClick={(e) => {
                    setOpen(false);
                    handleNavClick(e, item.href);
                  }}
                  aria-current={active ? "page" : undefined}
                  className={cn(
                    "block rounded-xl px-4 py-3 text-[0.9375rem] transition-colors duration-150",
                    active ? "bg-accent font-medium text-primary" : "text-foreground hover:bg-accent",
                  )}
                >
                  {item.label}
                </Link>
              </li>
            );
          })}
        </ul>
        <div className="flex flex-col gap-3 border-t border-border p-4">
          {signedIn ? (
            <Link
              href="/app/overview"
              onClick={() => setOpen(false)}
              className="flex h-11 w-full items-center justify-center gap-1.5 rounded-full bg-primary text-sm font-medium text-primary-foreground"
            >
              <LayoutDashboard className="size-4" aria-hidden />
              Dashboard
            </Link>
          ) : (
            <>
              <Link
                href="/signin"
                onClick={() => setOpen(false)}
                className="flex h-11 w-full items-center justify-center rounded-full border border-border-strong text-sm font-medium"
              >
                Sign in
              </Link>
              <Link
                href="/signup"
                onClick={() => setOpen(false)}
                className="flex h-11 w-full items-center justify-center rounded-full bg-primary text-sm font-medium text-primary-foreground"
              >
                Get started
              </Link>
            </>
          )}
          <div className="flex items-center justify-between pt-1 text-sm text-muted-foreground">
            Theme
            <ThemeToggle />
          </div>
        </div>
      </nav>
    </>
  );
}
