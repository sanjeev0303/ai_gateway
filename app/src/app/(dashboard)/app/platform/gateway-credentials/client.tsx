"use client";

import * as React from "react";
import { Copy, MoreHorizontal, Plus } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { ActionDialog, FormDialog, OneTimeSecretDialog } from "@/components/dashboard";
import { toast } from "sonner";
import {
  createGatewayCredential,
  revokeGatewayCredential,
  rotateGatewayCredential,
  expireGatewayCredential,
  restoreGatewayCredential,
  type CreateCredentialState,
} from "./actions";

export function GatewayCredentialActions({ credential }: { credential: any }) {
  const [revokeOpen, setRevokeOpen] = React.useState(false);
  const [rotateOpen, setRotateOpen] = React.useState(false);
  const [expireOpen, setExpireOpen] = React.useState(false);
  const [restoreOpen, setRestoreOpen] = React.useState(false);
  const [secret, setSecret] = React.useState<string | null>(null);

  const isActive = credential.status === "active";
  const isExpired = credential.status === "expired";
  const isRevoked = credential.status === "revoked";

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
          {isActive && (
            <>
              <DropdownMenuItem onClick={() => setRotateOpen(true)}>Rotate</DropdownMenuItem>
              <DropdownMenuItem onClick={() => setExpireOpen(true)}>Expire immediately</DropdownMenuItem>
              <DropdownMenuSeparator />
            </>
          )}
          {isExpired && (
            <>
              <DropdownMenuItem onClick={() => setRestoreOpen(true)}>Restore</DropdownMenuItem>
              <DropdownMenuSeparator />
            </>
          )}
          {!isRevoked && (
            <DropdownMenuItem
              className="text-destructive focus:text-destructive"
              onClick={() => setRevokeOpen(true)}
            >
              Revoke
            </DropdownMenuItem>
          )}
        </DropdownMenuContent>
      </DropdownMenu>

      <ActionDialog
        open={revokeOpen}
        onOpenChange={setRevokeOpen}
        title="Revoke Credential"
        description={`Are you sure you want to revoke ${credential.name}? This action cannot be undone and will immediately break any workloads using this credential.`}
        action={async (state, formData) => {
          formData.set("id", String(credential.id));
          return revokeGatewayCredential(state, formData);
        }}
        submitLabel="Revoke"
        submitVariant="destructive"
        onSuccess={() => toast.success("Credential revoked")}
      />

      <ActionDialog
        open={expireOpen}
        onOpenChange={setExpireOpen}
        title="Expire Credential"
        description={`Are you sure you want to expire ${credential.name}? It will become unusable, but can be restored.`}
        action={async (state, formData) => {
          formData.set("id", String(credential.id));
          return expireGatewayCredential(state, formData);
        }}
        submitLabel="Expire"
        submitVariant="destructive"
        onSuccess={() => toast.success("Credential expired")}
      />

      <ActionDialog
        open={restoreOpen}
        onOpenChange={setRestoreOpen}
        title="Restore Credential"
        description={`Restore ${credential.name}? It will become active again.`}
        action={async (state, formData) => {
          formData.set("id", String(credential.id));
          return restoreGatewayCredential(state, formData);
        }}
        submitLabel="Restore"
        onSuccess={() => toast.success("Credential restored")}
      />

      <ActionDialog
        open={rotateOpen && !secret}
        onOpenChange={setRotateOpen}
        title="Rotate Credential"
        description={`Rotating ${credential.name} will invalidate the current secret and issue a new one.`}
        action={async (state, formData) => {
          formData.set("id", String(credential.id));
          return rotateGatewayCredential(state as CreateCredentialState, formData);
        }}
        submitLabel="Rotate"
        onSuccess={(state: unknown) => {
          const result = state as CreateCredentialState;
          if (result?.secret) setSecret(result.secret);
        }}
      />

      <OneTimeSecretDialog
        open={!!secret}
        onOpenChange={(open) => {
          if (!open) {
            setSecret(null);
            setRotateOpen(false);
          }
        }}
        secret={secret}
      />
    </>
  );
}

export function CreateCredentialButton() {
  const [open, setOpen] = React.useState(false);
  const [secret, setSecret] = React.useState<string | null>(null);

  if (secret) {
    return (
      <OneTimeSecretDialog
        open={true}
        onOpenChange={(isOpen) => {
          if (!isOpen) {
            setSecret(null);
            setOpen(false);
          }
        }}
        secret={secret}
        title="Credential Created"
        description="Copy your new secret key now. It will not be shown again. It cannot be recovered later."
      />
    );
  }

  return (
    <>
      <Button onClick={() => setOpen(true)} size="sm">
        <Plus className="mr-2 h-4 w-4" />
        New credential
      </Button>

      <FormDialog
        open={open}
        onOpenChange={setOpen}
        title="Create Gateway Credential"
        description="Issue a new key for workloads to call the gateway."
        action={createGatewayCredential}
        submitLabel="Create"
        onSuccess={(state: unknown) => {
          const result = state as CreateCredentialState;
          if (result?.secret) setSecret(result.secret);
        }}
      >
        <div className="grid gap-4">
          <div className="grid gap-2">
            <Label htmlFor="name">Name</Label>
            <Input id="name" name="name" placeholder="e.g. Production API" required />
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div className="grid gap-2">
              <Label htmlFor="subject_type">Subject Type</Label>
              <Input id="subject_type" name="subject_type" placeholder="e.g. service_account" required />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="subject_id">Subject ID</Label>
              <Input id="subject_id" name="subject_id" required />
            </div>
          </div>
          <div className="grid gap-2">
            <Label htmlFor="scopes">Scopes (comma-separated)</Label>
            <Input id="scopes" name="scopes" placeholder="e.g. models.invoke, audit.read" />
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div className="grid gap-2">
              <Label htmlFor="team_id">Team ID (Optional)</Label>
              <Input id="team_id" name="team_id" placeholder="Optional" />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="project_id">Project ID (Optional)</Label>
              <Input id="project_id" name="project_id" placeholder="Optional" />
            </div>
          </div>
          <div className="grid gap-2">
            <Label htmlFor="expiry_days">Expiry (days)</Label>
            <Input id="expiry_days" name="expiry_days" type="number" placeholder="Optional" min={1} />
          </div>
        </div>
      </FormDialog>
    </>
  );
}
