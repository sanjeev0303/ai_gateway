"use client";

import * as React from "react";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

interface ConfirmOptions {
  title: string;
  description: string;
  /** Names the action: "Delete budget", never "OK". */
  confirmLabel: string;
  destructive?: boolean;
}

interface Pending extends ConfirmOptions {
  resolve: (confirmed: boolean) => void;
}

let show: ((pending: Pending) => void) | null = null;

/** Asks before something hard to undo. Resolves true only on an explicit yes. */
export function confirmAction(options: ConfirmOptions): Promise<boolean> {
  return new Promise((resolve) => {
    if (!show) {
      resolve(false);
      return;
    }
    show({ ...options, resolve });
  });
}

/** Mounted once in the dashboard shell; serves every confirmAction call. */
export function ConfirmHost() {
  const [pending, setPending] = React.useState<Pending | null>(null);

  React.useEffect(() => {
    show = setPending;
    return () => {
      show = null;
    };
  }, []);

  const settle = (confirmed: boolean) => {
    pending?.resolve(confirmed);
    setPending(null);
  };

  return (
    <Dialog open={pending !== null} onOpenChange={(open) => !open && settle(false)}>
      {pending && (
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{pending.title}</DialogTitle>
            <DialogDescription>{pending.description}</DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => settle(false)}>
              Cancel
            </Button>
            <Button variant={pending.destructive ? "destructive" : "default"} onClick={() => settle(true)}>
              {pending.confirmLabel}
            </Button>
          </DialogFooter>
        </DialogContent>
      )}
    </Dialog>
  );
}
