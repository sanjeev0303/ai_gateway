"use client";

import * as React from "react";
import { Menu } from "lucide-react";

import { SidebarNav } from "./sidebar";
import { Button } from "@ai-governance/console-ui/ui/button";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetTitle,
  SheetTrigger,
} from "@ai-governance/console-ui/ui/sheet";

/** Hamburger + slide-over that hosts the shared sidebar on small screens. */
export function MobileNav() {
  const [open, setOpen] = React.useState(false);

  return (
    <Sheet open={open} onOpenChange={setOpen}>
      <SheetTrigger asChild>
        <Button
          variant="ghost"
          size="icon"
          className="lg:hidden"
          aria-label="Open navigation"
        >
          <Menu className="size-5" aria-hidden />
        </Button>
      </SheetTrigger>
      <SheetContent side="left" className="w-72 gap-0 bg-background p-0">
        {/* Radix needs these for the dialog's accessible name. */}
        <SheetTitle className="sr-only">Dashboard navigation</SheetTitle>
        <SheetDescription className="sr-only">
          Jump to a section of the Tollseal dashboard.
        </SheetDescription>
        <SidebarNav onNavigate={() => setOpen(false)} />
      </SheetContent>
    </Sheet>
  );
}
