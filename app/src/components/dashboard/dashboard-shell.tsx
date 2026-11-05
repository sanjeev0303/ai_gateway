"use client";

import { createContext, useContext, useEffect, useState } from "react";
import { SidebarNav } from "@/components/dashboard/sidebar";
import { Topbar } from "@/components/dashboard/topbar";
import { ConfirmHost } from "@/components/dashboard/confirm-dialog";
import { cn } from "@/lib/utils";

interface SidebarContextValue {
  collapsed: boolean;
  setCollapsed: (collapsed: boolean) => void;
  toggleCollapsed: () => void;
}

const SidebarContext = createContext<SidebarContextValue>({
  collapsed: false,
  setCollapsed: () => {},
  toggleCollapsed: () => {},
});

export const useSidebar = () => useContext(SidebarContext);

const STORAGE_KEY = "sentinel_sidebar_collapsed";

export function DashboardShell({ children }: { children: React.ReactNode }) {
  const [collapsed, setCollapsed] = useState(false);

  useEffect(() => {
    try {
      const stored = localStorage.getItem(STORAGE_KEY);
      if (stored !== null) {
        setCollapsed(stored === "true");
      }
    } catch {
      // localStorage may be disabled in restricted environments
    }
  }, []);

  const toggleCollapsed = () => {
    setCollapsed((prev) => {
      const next = !prev;
      try {
        localStorage.setItem(STORAGE_KEY, String(next));
      } catch {
        // ignore storage errors
      }
      return next;
    });
  };

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      const target = e.target as HTMLElement | null;
      if (target && (target.tagName === "INPUT" || target.tagName === "TEXTAREA" || target.isContentEditable)) {
        return;
      }
      if (e.key === "[" && !e.metaKey && !e.ctrlKey && !e.altKey) {
        e.preventDefault();
        toggleCollapsed();
      }
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, []);

  return (
    <SidebarContext.Provider value={{ collapsed, setCollapsed, toggleCollapsed }}>
      <div className="relative flex h-screen overflow-hidden bg-background">
        <a
          href="#dashboard-main"
          className="sr-only focus:not-sr-only focus:absolute focus:top-3 focus:left-3 focus:z-50 focus:rounded-full focus:bg-primary focus:px-4 focus:py-2 focus:text-sm focus:text-primary-foreground"
        >
          Skip to content
        </a>
        {/* Iris light behind the top of the console, for the glass to catch */}
        <div aria-hidden className="ambient pointer-events-none absolute inset-x-0 top-0 h-[520px] opacity-80" />
        {/* The sidebar floats as a glass panel, the heavier material of the navigation layer */}
        <aside
          className={cn(
            "relative hidden shrink-0 py-3 pl-3 transition-[width] duration-200 ease-(--ease-out-strong) lg:block",
            collapsed ? "w-[5.5rem]" : "w-[17.5rem]",
          )}
        >
          <div className="liquid-glass-panel h-full overflow-hidden rounded-2xl border">
            <SidebarNav collapsed={collapsed} onToggleCollapse={toggleCollapsed} />
          </div>
        </aside>

        <div className="relative flex min-w-0 flex-1 flex-col overflow-hidden">
          {/* The top bar floats inside the scroll area, so content passes under its glass */}
          <main id="dashboard-main" tabIndex={-1} className="min-h-0 flex-1 overflow-y-auto page-scrollbar outline-none">
            <Topbar />
            <div className="mx-auto w-full max-w-[1600px] lg:px-2">{children}</div>
          </main>
        </div>
      </div>
      <ConfirmHost />
    </SidebarContext.Provider>
  );
}
