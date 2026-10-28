"use client";

import { useEffect, useState, useSyncExternalStore } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { Moon, Sun, Monitor, Menu, X } from "lucide-react";
import { useTheme } from "next-themes";
import { Logo } from "@/components/logo";
import { MARKETING_NAV } from "@/lib/constants";
import { cn } from "@/lib/utils";
import { useLenis } from "@/components/providers/lenis-provider";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

const emptySubscribe = () => () => {};

function ThemeToggle() {
  const { theme, resolvedTheme, setTheme } = useTheme();
  const mounted = useSyncExternalStore(emptySubscribe, () => true, () => false);

  if (!mounted) return (
    <button
      className="size-8 flex items-center justify-center focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[#6F4E37]"
      aria-label="Toggle theme"
    >
      <Sun className="size-4 opacity-40" aria-hidden />
    </button>
  );

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          className="size-8 flex items-center justify-center text-[#7A6458] transition-colors duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[#6F4E37]"
          aria-label="Toggle theme"
        >
          {resolvedTheme === "dark" ? <Moon className="size-4" aria-hidden /> : <Sun className="size-4" aria-hidden />}
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent
        align="end"
        className="w-32 rounded-none border-[rgba(111,78,55,0.16)] bg-canvas"
      >
        <DropdownMenuLabel className="text-xs font-medium text-[#7A6458]" style={{ fontFamily: "var(--font-fira-code)" }}>
          Theme
        </DropdownMenuLabel>
        <DropdownMenuItem onClick={() => setTheme("light")} className="text-xs text-[#4A3A30]">
          <Sun className="mr-2 size-3.5" aria-hidden /> Light
        </DropdownMenuItem>
        <DropdownMenuItem onClick={() => setTheme("dark")} className="text-xs text-[#4A3A30]">
          <Moon className="mr-2 size-3.5" aria-hidden /> Dark
        </DropdownMenuItem>
        <DropdownMenuItem onClick={() => setTheme("system")} className="text-xs text-[#4A3A30]">
          <Monitor className="mr-2 size-3.5" aria-hidden /> System
          {theme === "system" && <span className="ml-auto text-xs text-[#7A6458]">active</span>}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

export function MarketingNav({ className }: { className?: string }) {
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

  const handleNavClick = (e: React.MouseEvent<HTMLAnchorElement>, href: string) => {
    if (href.startsWith("/#")) {
      const targetId = href.replace("/#", "");
      if (pathname === "/") {
        e.preventDefault();
        const element = document.getElementById(targetId);
        if (element) {
          scrollTo(element, { offset: -70 });
          window.history.pushState(null, "", `#${targetId}`);
        }
      }
    }
  };

  return (
    <>
      <header
        className={cn(
          "sticky top-0 z-50 transition-all duration-300",
          scrolled
            ? "border-b border-[rgba(111,78,55,0.12)] backdrop-blur-sm"
            : "border-b border-transparent",
          className,
        )}
        style={{ background: scrolled ? "rgba(229,228,226,0.94)" : "var(--canvas)" }}
      >
        <div className="mx-auto flex h-14 max-w-7xl items-center gap-8 px-6 sm:px-8">
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
            className="shrink-0 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[#6F4E37]"
          >
            <Logo />
          </Link>

          {/* Desktop nav */}
          <nav aria-label="Main" className="hidden md:block ml-4">
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
                      className="block px-4 py-2 text-sm transition-colors duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[#6F4E37]"
                      style={{
                        color: active ? "#6F4E37" : "#4A3A30",
                        fontWeight: active ? 600 : 400,
                        letterSpacing: "-0.01em",
                      }}
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
            <ThemeToggle />

            <Link
              href="/app/overview"
              id="nav-sign-in"
              className="hidden sm:inline-flex items-center h-8 px-4 text-sm font-medium text-[#4A3A30] transition-colors duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[#6F4E37]"
              style={{ letterSpacing: "-0.01em" }}
            >
              Sign in
            </Link>

            <Link
              href="/app/overview"
              id="nav-open-dashboard"
              className="inline-flex items-center h-8 px-4 text-sm font-semibold text-[#F5F3F0] transition-colors duration-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[#6F4E37]"
              style={{ background: "#6F4E37", letterSpacing: "-0.01em" }}
              onMouseEnter={e => (e.currentTarget.style.background = "#4A3224")}
              onMouseLeave={e => (e.currentTarget.style.background = "#6F4E37")}
            >
              Open dashboard
            </Link>

            {/* Mobile menu toggle */}
            <button
              type="button"
              onClick={() => setOpen(!open)}
              className="flex size-8 items-center justify-center text-[#4A3A30] border border-[rgba(111,78,55,0.20)] md:hidden focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[#6F4E37]"
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
          className="fixed inset-0 z-40 md:hidden"
          style={{ background: "rgba(28,20,18,0.5)" }}
          onClick={() => setOpen(false)}
          aria-hidden
        />
      )}
      <nav
        aria-label="Mobile"
        className={cn(
          "fixed top-14 right-0 bottom-0 z-50 w-72 overflow-y-auto transition-transform duration-300 ease-out md:hidden",
          open ? "translate-x-0" : "translate-x-full"
        )}
        style={{ background: "var(--canvas)", borderLeft: "1px solid rgba(111,78,55,0.12)" }}
      >
        <ul className="p-4 space-y-0">
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
                  className="block px-4 py-3 text-sm font-medium border-b border-[rgba(111,78,55,0.08)] transition-colors duration-150"
                  style={{
                    color: active ? "#6F4E37" : "#4A3A30",
                    fontWeight: active ? 600 : 400,
                    letterSpacing: "-0.01em",
                  }}
                >
                  {item.label}
                </Link>
              </li>
            );
          })}
        </ul>
        <div className="border-t border-[rgba(111,78,55,0.10)] p-4 space-y-3">
          <Link
            href="/app/overview"
            onClick={() => setOpen(false)}
            className="flex items-center justify-center w-full h-10 text-sm font-medium text-[#4A3A30] border border-[rgba(111,78,55,0.24)]"
          >
            Sign in
          </Link>
          <Link
            href="/app/overview"
            onClick={() => setOpen(false)}
            className="flex items-center justify-center w-full h-10 text-sm font-semibold text-[#F5F3F0]"
            style={{ background: "#6F4E37" }}
          >
            Open dashboard
          </Link>
        </div>
      </nav>
    </>
  );
}
