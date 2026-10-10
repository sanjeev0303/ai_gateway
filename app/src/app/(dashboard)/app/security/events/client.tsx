"use client";

import { useState } from "react";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";

export function SecurityEventActions({ event }: { event: any }) {
  const [open, setOpen] = useState(false);

  return (
    <>
      <Button variant="outline" size="sm" onClick={() => setOpen(true)}>View details</Button>
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>Security Event Details</DialogTitle>
          </DialogHeader>
          <div className="py-4 space-y-4 max-h-[70vh] overflow-y-auto">
            <div className="grid grid-cols-2 gap-4">
              <div>
                <div className="text-sm font-semibold text-muted-foreground">Title</div>
                <div>{event.title}</div>
              </div>
              <div>
                <div className="text-sm font-semibold text-muted-foreground">Category</div>
                <div>{event.category}</div>
              </div>
              <div>
                <div className="text-sm font-semibold text-muted-foreground">Severity</div>
                <div>{event.severity}</div>
              </div>
              <div>
                <div className="text-sm font-semibold text-muted-foreground">Source</div>
                <div>{event.source}</div>
              </div>
              <div>
                <div className="text-sm font-semibold text-muted-foreground">Occurred At</div>
                <div>{new Date(event.created_at).toLocaleString()}</div>
              </div>
            </div>

            <div className="mt-6">
              <div className="text-sm font-semibold text-muted-foreground mb-2">Detail</div>
              <pre className="bg-muted p-4 rounded-md text-sm whitespace-pre-wrap">
                {event.detail ? JSON.stringify(event.detail, null, 2) : "None"}
              </pre>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setOpen(false)}>Close</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
