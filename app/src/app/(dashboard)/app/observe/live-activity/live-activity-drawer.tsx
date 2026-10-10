"use client";

import * as React from "react";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";
import { Button } from "@/components/ui/button";

export interface LiveRow {
  id: string;
  href?: string;
  when: string;
  whenTs: number;
  requestId: string;
  who: string;
  project: string;
  team: string;
  model: string;
  provider: string;
  state: string;
  stateTone?: any;
  status: string;
  statusTone?: any;
}

export function LiveActivityDrawer({ row }: { row: LiveRow }) {
  return (
    <Sheet>
      <SheetTrigger asChild>
        <Button variant="ghost" size="sm">
          View
        </Button>
      </SheetTrigger>
      <SheetContent>
        <SheetHeader>
          <SheetTitle>Request Detail</SheetTitle>
          <SheetDescription>
            Request ID: {row.requestId}
          </SheetDescription>
        </SheetHeader>
        <div className="mt-6 space-y-4">
          <div>
            <h4 className="text-sm font-medium text-muted-foreground">Who</h4>
            <p className="text-sm font-semibold">{row.who}</p>
          </div>
          <div>
            <h4 className="text-sm font-medium text-muted-foreground">Provider</h4>
            <p className="text-sm font-semibold">{row.provider}</p>
          </div>
          <div>
            <h4 className="text-sm font-medium text-muted-foreground">Model</h4>
            <p className="text-sm font-semibold">{row.model}</p>
          </div>
          <div>
            <h4 className="text-sm font-medium text-muted-foreground">Project</h4>
            <p className="text-sm font-semibold">{row.project}</p>
          </div>
          <div>
            <h4 className="text-sm font-medium text-muted-foreground">Team</h4>
            <p className="text-sm font-semibold">{row.team}</p>
          </div>
          <div>
            <h4 className="text-sm font-medium text-muted-foreground">State</h4>
            <p className="text-sm font-semibold capitalize">{row.state}</p>
          </div>
          <div>
            <h4 className="text-sm font-medium text-muted-foreground">Status</h4>
            <p className="text-sm font-semibold capitalize">{row.status}</p>
          </div>
          <div>
            <h4 className="text-sm font-medium text-muted-foreground">When</h4>
            <p className="text-sm font-semibold">{row.when}</p>
          </div>
        </div>
      </SheetContent>
    </Sheet>
  );
}
