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

import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

export interface CreateCredentialButtonProps {
  members?: Array<{ id: string; name: string; email: string }>;
  serviceAccounts?: Array<{ id: string; name: string }>;
  teams?: Array<{ id: string; name: string }>;
  projects?: Array<{ id: string; name: string }>;
  currentUserId?: string;
}

export function CreateCredentialButton({
  members = [],
  serviceAccounts = [],
  teams = [],
  projects = [],
  currentUserId,
}: CreateCredentialButtonProps) {
  const [open, setOpen] = React.useState(false);
  const [secret, setSecret] = React.useState<string | null>(null);

  const [subjectType, setSubjectType] = React.useState<"user" | "service_account">("user");
  const [subjectId, setSubjectId] = React.useState(
    currentUserId || members[0]?.id || serviceAccounts[0]?.id || ""
  );
  const [teamId, setTeamId] = React.useState<string>("none");
  const [projectId, setProjectId] = React.useState<string>("none");
  const [expiryOption, setExpiryOption] = React.useState<string>("90");

  React.useEffect(() => {
    if (subjectType === "user") {
      setSubjectId(currentUserId || members[0]?.id || "");
    } else {
      setSubjectId(serviceAccounts[0]?.id || "");
    }
  }, [subjectType, currentUserId, members, serviceAccounts]);

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
        title="Gateway Credential Created"
        description="Copy your new secret key now. It will not be shown again and cannot be recovered."
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
        description="Issue a new API key for callers to invoke models through the Sentinel gateway."
        action={createGatewayCredential}
        submitLabel="Create Key"
        onSuccess={(state: unknown) => {
          const result = state as CreateCredentialState;
          if (result?.secret) setSecret(result.secret);
        }}
      >
        <div className="grid gap-4 py-2">
          <div className="grid gap-2">
            <Label htmlFor="name">Credential Name</Label>
            <Input id="name" name="name" placeholder="e.g. My Dev Key, Ingestion Worker" required />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="grid gap-2">
              <Label>Identity Type</Label>
              <Select
                value={subjectType}
                onValueChange={(val: "user" | "service_account") => setSubjectType(val)}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Select type" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="user">Human Employee (User)</SelectItem>
                  <SelectItem value="service_account">Service Account (Machine)</SelectItem>
                </SelectContent>
              </Select>
              <input type="hidden" name="subject_type" value={subjectType} />
            </div>

            <div className="grid gap-2">
              <Label>Subject Identity</Label>
              {subjectType === "user" ? (
                members.length > 0 ? (
                  <Select value={subjectId} onValueChange={setSubjectId}>
                    <SelectTrigger>
                      <SelectValue placeholder="Select member" />
                    </SelectTrigger>
                    <SelectContent>
                      {members.map((m) => (
                        <SelectItem key={m.id} value={m.id}>
                          {m.name} ({m.email})
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                ) : (
                  <Input
                    name="subject_id"
                    value={subjectId}
                    onChange={(e) => setSubjectId(e.target.value)}
                    placeholder="User ID UUID"
                    required
                  />
                )
              ) : serviceAccounts.length > 0 ? (
                <Select value={subjectId} onValueChange={setSubjectId}>
                  <SelectTrigger>
                    <SelectValue placeholder="Select service account" />
                  </SelectTrigger>
                  <SelectContent>
                    {serviceAccounts.map((sa) => (
                      <SelectItem key={sa.id} value={sa.id}>
                        {sa.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              ) : (
                <Input
                  name="subject_id"
                  value={subjectId}
                  onChange={(e) => setSubjectId(e.target.value)}
                  placeholder="Service Account UUID"
                  required
                />
              )}
              <input type="hidden" name="subject_id" value={subjectId} />
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="grid gap-2">
              <Label>Team Scope (Optional)</Label>
              <Select value={teamId} onValueChange={setTeamId}>
                <SelectTrigger>
                  <SelectValue placeholder="Organization-wide" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="none">None (Organization-wide)</SelectItem>
                  {teams.map((t) => (
                    <SelectItem key={t.id} value={t.id}>
                      {t.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <input type="hidden" name="team_id" value={teamId === "none" ? "" : teamId} />
            </div>

            <div className="grid gap-2">
              <Label>Project Scope (Optional)</Label>
              <Select value={projectId} onValueChange={setProjectId}>
                <SelectTrigger>
                  <SelectValue placeholder="All projects" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="none">None (All projects)</SelectItem>
                  {projects.map((p) => (
                    <SelectItem key={p.id} value={p.id}>
                      {p.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <input type="hidden" name="project_id" value={projectId === "none" ? "" : projectId} />
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="grid gap-2">
              <Label htmlFor="scopes">Permitted Scopes</Label>
              <Input
                id="scopes"
                name="scopes"
                defaultValue="gateway"
                placeholder="e.g. gateway"
                required
              />
            </div>

            <div className="grid gap-2">
              <Label>Expiration</Label>
              <Select value={expiryOption} onValueChange={setExpiryOption}>
                <SelectTrigger>
                  <SelectValue placeholder="Select expiry" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="30">30 days</SelectItem>
                  <SelectItem value="60">60 days</SelectItem>
                  <SelectItem value="90">90 days</SelectItem>
                  <SelectItem value="365">1 year (365 days)</SelectItem>
                  <SelectItem value="never">Never expires</SelectItem>
                </SelectContent>
              </Select>
              <input
                type="hidden"
                name="expiry_days"
                value={expiryOption === "never" ? "" : expiryOption}
              />
            </div>
          </div>
        </div>
      </FormDialog>
    </>
  );
}

