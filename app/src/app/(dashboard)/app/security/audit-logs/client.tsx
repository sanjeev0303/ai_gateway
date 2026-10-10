"use client";

import { useState } from "react";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";

export function AuditLogActions({ record }: { record: any }) {
  const [open, setOpen] = useState(false);

  return (
    <>
      <Button variant="outline" size="sm" onClick={() => setOpen(true)}>View details</Button>
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>Audit Record Details</DialogTitle>
          </DialogHeader>
          <div className="py-4 space-y-4 max-h-[70vh] overflow-y-auto">
            <div className="grid grid-cols-2 gap-4">
              <div>
                <div className="text-sm font-semibold text-muted-foreground">Action</div>
                <div>{record.action}</div>
              </div>
              <div>
                <div className="text-sm font-semibold text-muted-foreground">Occurred at</div>
                <div>{new Date(record.created_at).toLocaleString()}</div>
              </div>
              <div>
                <div className="text-sm font-semibold text-muted-foreground">Actor Type</div>
                <div>{record.actor_type}</div>
              </div>
              <div>
                <div className="text-sm font-semibold text-muted-foreground">Actor ID</div>
                <div className="break-all">{record.actor_user_id || record.actor_service_account_id || "N/A"}</div>
              </div>
              <div>
                <div className="text-sm font-semibold text-muted-foreground">Resource Type</div>
                <div>{record.resource_type}</div>
              </div>
              <div>
                <div className="text-sm font-semibold text-muted-foreground">Resource ID</div>
                <div className="break-all">{record.resource_id || "N/A"}</div>
              </div>
            </div>

            <div className="grid grid-cols-2 gap-4 mt-6">
              <div>
                <div className="text-sm font-semibold text-muted-foreground mb-2">Before</div>
                <pre className="bg-muted p-4 rounded-md text-sm whitespace-pre-wrap">
                  {record.before ? JSON.stringify(record.before, null, 2) : "None"}
                </pre>
              </div>
              <div>
                <div className="text-sm font-semibold text-muted-foreground mb-2">After</div>
                <pre className="bg-muted p-4 rounded-md text-sm whitespace-pre-wrap">
                  {record.after ? JSON.stringify(record.after, null, 2) : "None"}
                </pre>
              </div>
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
