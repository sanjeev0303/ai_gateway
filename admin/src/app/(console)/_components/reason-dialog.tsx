"use client";

import * as React from "react";

import { Spinner } from "@ai-governance/console-ui/dashboard/state";
import { Button } from "@ai-governance/console-ui/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@ai-governance/console-ui/ui/dialog";
import { Label } from "@ai-governance/console-ui/ui/label";
import { Textarea } from "@ai-governance/console-ui/ui/textarea";
import { showError, showSuccess } from "@ai-governance/console-ui/lib/toast";

const MAX_REASON = 500;

interface ReasonDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  description: string;
  /** Names the action: "Suspend organization", never "OK". */
  confirmLabel: string;
  destructive?: boolean;
  /** Extra fields above the reason, such as the plan to move to. */
  children?: React.ReactNode;
  /** What is still missing from the extra fields, shown when confirm is pressed. */
  blocker?: string;
  successMessage: string;
  onConfirm: (reason: string) => Promise<{ error?: string }>;
}

/**
 * Every platform action asks why. The reason is required by the control API
 * and lands in the platform audit log and in the organization's own log.
 */
export function ReasonDialog({
  open,
  onOpenChange,
  title,
  description,
  confirmLabel,
  destructive = false,
  children,
  blocker,
  successMessage,
  onConfirm,
}: ReasonDialogProps) {
  const [reason, setReason] = React.useState("");
  const [error, setError] = React.useState<string | null>(null);
  const [pending, startTransition] = React.useTransition();
  const fieldId = React.useId();
  const fieldRef = React.useRef<HTMLTextAreaElement>(null);

  const trimmed = reason.trim();

  const close = (next: boolean) => {
    if (pending) return;
    if (!next) {
      setReason("");
      setError(null);
    }
    onOpenChange(next);
  };

  const submit = (event: React.FormEvent) => {
    event.preventDefault();
    if (pending) return;
    // The button stays enabled; a missing input is explained, not silently blocked.
    if (blocker) {
      setError(blocker);
      return;
    }
    if (trimmed.length === 0) {
      setError("Give a reason for the audit log.");
      fieldRef.current?.focus();
      return;
    }
    setError(null);
    startTransition(async () => {
      const result = await onConfirm(trimmed);
      if (result.error) {
        setError(result.error);
        showError(result.error);
        return;
      }
      showSuccess(successMessage);
      setReason("");
      onOpenChange(false);
    });
  };

  return (
    <Dialog open={open} onOpenChange={close}>
      <DialogContent>
        <form onSubmit={submit} className="flex flex-col gap-5">
          <DialogHeader>
            <DialogTitle>{title}</DialogTitle>
            <DialogDescription>{description}</DialogDescription>
          </DialogHeader>

          {children}

          <div className="flex flex-col gap-2">
            <Label htmlFor={fieldId}>Reason</Label>
            <Textarea
              ref={fieldRef}
              id={fieldId}
              name="reason"
              autoComplete="off"
              value={reason}
              onChange={(event) => setReason(event.target.value)}
              placeholder="Chargeback on the March invoice…"
              rows={3}
              maxLength={MAX_REASON}
              aria-invalid={error ? true : undefined}
              aria-describedby={error ? `${fieldId}-error` : undefined}
              disabled={pending}
            />
            {error ? (
              <p id={`${fieldId}-error`} role="alert" className="text-sm text-rose-700 dark:text-rose-400">
                {error}
              </p>
            ) : (
              <p className="text-xs text-muted-foreground">
                Recorded in the audit log, where the organization’s owners can read it.{" "}
                <span className="tabular-nums">
                  {trimmed.length}/{MAX_REASON}
                </span>
              </p>
            )}
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => close(false)} disabled={pending}>
              Cancel
            </Button>
            <Button type="submit" variant={destructive ? "destructive" : "default"} disabled={pending}>
              {pending && <Spinner />}
              {confirmLabel}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
