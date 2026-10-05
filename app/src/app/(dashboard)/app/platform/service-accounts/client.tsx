"use client";

import * as React from "react";
import { MoreHorizontal, Plus } from "lucide-react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { ActionDialog, FormDialog } from "@/components/dashboard/dialogs";
import { createServiceAccount, removeServiceAccount } from "./actions";

export function ServiceAccountActions({ account }: { account: Record<string, unknown> }) {
  const [removeOpen, setRemoveOpen] = React.useState(false);
  const isRevoked = account.status === "revoked";

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="ghost" className="h-8 w-8 p-0">
            <span className="sr-only">Open menu</span>
            <MoreHorizontal className="h-4 w-4" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          {!isRevoked && (
            <DropdownMenuItem
              className="text-destructive focus:text-destructive"
              onClick={() => setRemoveOpen(true)}
            >
              Remove
            </DropdownMenuItem>
          )}
        </DropdownMenuContent>
      </DropdownMenu>

      <ActionDialog
        open={removeOpen}
        onOpenChange={setRemoveOpen}
        title="Remove Service Account"
        description={`Are you sure you want to remove ${account.name}? It cannot be undone.`}
        action={async (state, formData) => {
          formData.set("id", String(account.id));
          return removeServiceAccount(state, formData);
        }}
        submitLabel="Remove"
        submitVariant="destructive"
        onSuccess={() => toast.success("Service account removed")}
      />
    </>
  );
}

export function CreateServiceAccountButton() {
  const [open, setOpen] = React.useState(false);

  return (
    <>
      <Button onClick={() => setOpen(true)} size="sm">
        <Plus className="mr-2 h-4 w-4" />
        New service account
      </Button>

      <FormDialog
        open={open}
        onOpenChange={setOpen}
        title="Create Service Account"
        description="Issue a new identity for automated workloads."
        action={createServiceAccount}
        submitLabel="Create"
        onSuccess={() => toast.success("Service account created")}
      >
        <div className="grid gap-4">
          <div className="grid gap-2">
            <Label htmlFor="name">Name</Label>
            <Input id="name" name="name" placeholder="e.g. CI Pipeline" required />
          </div>
          <div className="grid gap-2">
            <Label htmlFor="description">Description</Label>
            <Input id="description" name="description" placeholder="Optional" />
          </div>
        </div>
      </FormDialog>
    </>
  );
}
