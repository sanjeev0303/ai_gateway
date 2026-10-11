"use client";

import { useRouter } from "next/navigation";
import { RefreshCw } from "lucide-react";
import { Button } from "@/components/ui/button";
import { RangeTabs } from "@/components/dashboard/range-tabs";
import * as React from "react";
import { Suspense, useEffect, useState } from "react";

interface HeaderActionsProps {
  lastUpdated: Date;
}

export function HeaderActions({ lastUpdated }: HeaderActionsProps) {
  const router = useRouter();
  const [mounted, setMounted] = useState(false);

  useEffect(() => {
    setMounted(true);
  }, []);

  return (
    <div className="flex flex-wrap items-center gap-2">
      <Suspense fallback={null}>
        <RangeTabs />
      </Suspense>
      <Button variant="outline" size="sm" onClick={() => router.refresh()} aria-label="Refresh data">
        <RefreshCw className="size-4 mr-1.5" aria-hidden />
        Refresh
      </Button>
      {mounted && (
        <span
          suppressHydrationWarning
          className="hidden sm:inline-flex items-center gap-1.5 text-xs text-muted-foreground px-2 py-1 rounded-md bg-muted"
        >
          Updated {lastUpdated.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}
        </span>
      )}
    </div>
  );
}