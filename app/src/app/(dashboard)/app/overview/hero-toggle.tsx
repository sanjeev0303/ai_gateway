"use client";

import { useRouter } from "next/navigation";
import { ChevronDown, ChevronUp } from "lucide-react";

import { Button } from "@ai-governance/console-ui/ui/button";

const ONE_YEAR = 60 * 60 * 24 * 365;

/** Remembers the choice in a cookie, so the server renders the right size first. */
export function HeroToggle({ cookieName, collapsed }: { cookieName: string; collapsed: boolean }) {
  const router = useRouter();

  const toggle = () => {
    document.cookie = `${cookieName}=${collapsed ? "expanded" : "collapsed"}; path=/app; max-age=${ONE_YEAR}; samesite=lax`;
    router.refresh();
  };

  return (
    <Button
      type="button"
      variant="ghost"
      size="icon-sm"
      onClick={toggle}
      aria-expanded={!collapsed}
      aria-label={collapsed ? "Show the introduction" : "Hide the introduction"}
      title={collapsed ? "Show the introduction" : "Hide the introduction"}
    >
      {collapsed ? <ChevronDown /> : <ChevronUp />}
    </Button>
  );
}
