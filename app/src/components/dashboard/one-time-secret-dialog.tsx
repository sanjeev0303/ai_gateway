"use client";

import * as React from "react";
import { Copy } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { FormDialog } from "@/components/dashboard/dialogs";

export function OneTimeSecretDialog({
  open,
  onOpenChange,
  secret,
  title = "New Secret Generated",
  description = "Copy this secret now. It will not be shown again. It cannot be recovered later.",
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  secret: string | null;
  title?: string;
  description?: string;
}) {
  return (
    <FormDialog
      open={open && !!secret}
      onOpenChange={onOpenChange}
      title={title}
      description={description}
      action={async () => ({ success: true })}
      submitLabel="Done"
    >
      <div className="flex items-center gap-2">
        <Input readOnly value={secret || ""} />
        <Button
          type="button"
          variant="outline"
          size="icon"
          onClick={() => {
            if (secret) {
              navigator.clipboard.writeText(secret);
              toast.success("Copied to clipboard");
            }
          }}
        >
          <Copy className="h-4 w-4" />
        </Button>
      </div>
    </FormDialog>
  );
}
