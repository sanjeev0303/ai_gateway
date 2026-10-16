"use client";

import * as React from "react";
import { Loader2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { StatusBadge } from "@/components/dashboard/status-badge";

interface ActionState {
  error?: string;
  success?: boolean;
}

export interface ActionDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  description: React.ReactNode;
  action: (state: ActionState, formData: FormData) => Promise<ActionState>;
  submitLabel?: string;
  submitVariant?: "default" | "destructive";
  onSuccess?: (state?: any) => void;
}

function ActionDialogInner({
  title,
  description,
  action,
  submitLabel = "Confirm",
  submitVariant = "default",
  onOpenChange,
  onSuccess,
}: Omit<ActionDialogProps, "open">) {
  const [state, formAction, pending] = React.useActionState<ActionState, FormData>(
    action,
    {}
  );

  React.useEffect(() => {
    if (state.success && !pending) {
      onOpenChange(false);
      onSuccess?.(state);
    }
  }, [state.success, pending, onOpenChange, onSuccess]);

  return (
    <DialogContent>
      <DialogHeader>
        <DialogTitle>{title}</DialogTitle>
        <DialogDescription>{description}</DialogDescription>
      </DialogHeader>
      <form action={formAction}>
        <DialogFooter className="gap-2 sm:justify-between items-center w-full">
          <div className="flex-1">
            {state.error && <StatusBadge tone="danger">{state.error}</StatusBadge>}
          </div>
          <div className="flex gap-2">
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => onOpenChange(false)}
              disabled={pending}
            >
              Cancel
            </Button>
            <Button
              type="submit"
              variant={submitVariant}
              size="sm"
              disabled={pending}
            >
              {pending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              {submitLabel}
            </Button>
          </div>
        </DialogFooter>
      </form>
    </DialogContent>
  );
}

export function ActionDialog({ open, onOpenChange, ...props }: ActionDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      {open ? <ActionDialogInner onOpenChange={onOpenChange} {...props} /> : null}
    </Dialog>
  );
}

export interface FormDialogProps extends Omit<ActionDialogProps, "description" | "action" | "onSuccess"> {
  description?: React.ReactNode;
  children: React.ReactNode;
  action: (state: any, formData: FormData) => Promise<any>;
  onSuccess?: (state: any) => void;
}

function FormDialogInner({
  title,
  description,
  children,
  action,
  submitLabel = "Save",
  submitVariant = "default",
  onOpenChange,
  onSuccess,
}: Omit<FormDialogProps, "open">) {
  const [state, formAction, pending] = React.useActionState(action, {});

  React.useEffect(() => {
    if (state && typeof state === "object" && "success" in state && state.success && !pending) {
      onOpenChange(false);
      onSuccess?.(state);
    }
  }, [state, pending, onOpenChange, onSuccess]);

  return (
    <DialogContent>
      <DialogHeader>
        <DialogTitle>{title}</DialogTitle>
        {description && <DialogDescription>{description}</DialogDescription>}
      </DialogHeader>
      <form action={formAction} className="space-y-4">
        <div className="space-y-4 py-4">{children}</div>
        <DialogFooter className="gap-2 sm:justify-between items-center w-full">
          <div className="flex-1">
            {state?.error && <StatusBadge tone="danger">{state.error}</StatusBadge>}
          </div>
          <div className="flex gap-2">
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => onOpenChange(false)}
              disabled={pending}
            >
              Cancel
            </Button>
            <Button
              type="submit"
              variant={submitVariant}
              size="sm"
              disabled={pending}
            >
              {pending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              {submitLabel}
            </Button>
          </div>
        </DialogFooter>
      </form>
    </DialogContent>
  );
}

export function FormDialog({ open, onOpenChange, ...props }: FormDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      {open ? <FormDialogInner onOpenChange={onOpenChange} {...props} /> : null}
    </Dialog>
  );
}
