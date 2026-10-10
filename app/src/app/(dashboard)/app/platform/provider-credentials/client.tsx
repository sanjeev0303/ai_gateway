"use client";

import * as React from "react";
import { MoreHorizontal, Plus, RotateCcw, Trash2, Copy, Check } from "lucide-react";
import { DataTable, type Column } from "@/components/dashboard";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { toast } from "sonner";
import { createApiClient } from "@ai-governance/api-client";
import { CONTROL_API_BASE } from "@/lib/control-api";
import { useRouter } from "next/navigation";

interface ProviderCredential {
  id: string;
  name: string;
  provider_account_id: string;
  kind: string;
  fingerprint: string;
  tail: string;
  status: string;
  created_at: string;
  updated_at: string;
}

export interface CredentialRow {
  id: string;
  name: string;
  accountName: string;
  kind: string;
  fingerprint: string;
  tail: string;
  status: string;
  statusTone: "success" | "warning" | "danger" | "neutral";
  created: string;
}

const CREDENTIAL_STATUS_TONE: Record<string, "success" | "warning" | "danger" | "neutral"> = {
  active: "success",
  revoked: "neutral",
  expired: "danger",
};

function mapCredentialToRow(cred: ProviderCredential, accountName: string): CredentialRow {
  return {
    id: cred.id,
    name: cred.name,
    accountName,
    kind: cred.kind,
    fingerprint: cred.fingerprint,
    tail: cred.tail,
    status: cred.status,
    statusTone: CREDENTIAL_STATUS_TONE[cred.status] || "neutral",
    created: cred.created_at,
  };
}

const COLUMNS: readonly Column<CredentialRow>[] = [
  { key: "name", header: "Credential", primary: "name", secondary: "accountName", sort: "name" },
  { key: "kind", header: "Kind", primary: "kind", sort: "kind" },
  { key: "fingerprint", header: "Fingerprint", primary: "fingerprint", sort: "fingerprint" },
  { key: "tail", header: "Tail", primary: "tail", sort: "tail" },
  { key: "status", header: "Status", primary: "status", tone: "statusTone" },
  { key: "created", header: "Created", primary: "created", sort: "created", align: "right" },
];

export function ProviderCredentialsClient({ initialRows }: { initialRows: CredentialRow[] }) {
  const router = useRouter();
  const [rows, setRows] = React.useState<CredentialRow[]>(initialRows);
  const [isUpdating, setIsUpdating] = React.useState(false);

  // Add Credential
  const [addOpen, setAddOpen] = React.useState(false);
  const [addAccountId, setAddAccountId] = React.useState("");
  const [addName, setAddName] = React.useState("");
  const [addKind, setAddKind] = React.useState("openai");
  const [addSecret, setAddSecret] = React.useState("");

  // Rotate Credential
  const [rotateId, setRotateId] = React.useState<string | null>(null);
  const [rotateSecret, setRotateSecret] = React.useState("");

  // Revoke Credential
  const [revokeId, setRevokeId] = React.useState<string | null>(null);

  // Accounts for select
  const [accounts, setAccounts] = React.useState<{ id: string; name: string; provider_kind: string }[]>([]);

  const api = React.useMemo(() => createApiClient({ baseUrl: CONTROL_API_BASE }), []);

  React.useEffect(() => {
    // Fetch accounts for the add credential dialog
    api.providers.listAccounts().then((list) => {
      setAccounts(list.items);
    });
  }, [api]);

  const handleAddCredential = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      setIsUpdating(true);
      const cred = await api.providers.createCredential(addAccountId, {
        name: addName,
        kind: addKind,
        secret: addSecret,
      });
      const account = accounts.find(a => a.id === addAccountId);
      const accountName = account ? account.name : addAccountId;
      setRows((prev) => [mapCredentialToRow(cred, accountName), ...prev]);
      toast.success("Credential added");
      setAddOpen(false);
      setAddAccountId("");
      setAddName("");
      setAddSecret("");
    } catch (error) {
      toast.error("Failed to add credential");
    } finally {
      setIsUpdating(false);
    }
  };

  const handleRotateCredential = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!rotateId) return;
    try {
      setIsUpdating(true);
      const cred = await api.providers.rotateCredential(rotateId, { secret: rotateSecret });
      const account = accounts.find(a => a.id === cred.provider_account_id);
      const accountName = account ? account.name : cred.provider_account_id;
      setRows((prev) => prev.map((c) => c.id === rotateId ? mapCredentialToRow(cred, accountName) : c));
      toast.success("Credential rotated");
      setRotateId(null);
      setRotateSecret("");
    } catch (error) {
      toast.error("Failed to rotate credential");
    } finally {
      setIsUpdating(false);
    }
  };

  const handleRevokeCredential = async () => {
    if (!revokeId) return;
    try {
      setIsUpdating(true);
      await api.providers.removeCredential(revokeId);
      setRows((prev) => prev.filter((c) => c.id !== revokeId));
      toast.success("Credential revoked");
      setRevokeId(null);
    } catch (error) {
      toast.error("Failed to revoke credential");
    } finally {
      setIsUpdating(false);
    }
  };

  return (
    <>
      <DataTable
        columns={COLUMNS}
        rows={rows}
        rowKey="id"
        pageSize={10}
        emptyMessage="No credentials configured."
        toolbar={
          <div className="flex justify-end mb-4">
            <Button onClick={() => setAddOpen(true)} size="sm">
              <Plus className="mr-2 h-4 w-4" /> Add Credential
            </Button>
          </div>
        }
        actions={(row) => (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="icon" className="size-8 p-0" disabled={isUpdating}>
                <span className="sr-only">Open menu</span>
                <MoreHorizontal className="size-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuLabel>Actions</DropdownMenuLabel>
              {row.status === "active" && (
                <>
                  <DropdownMenuItem onClick={() => { setRotateId(row.id); setRotateSecret(""); }}>
                    <RotateCcw className="mr-2 size-4" /> Rotate
                  </DropdownMenuItem>
                  <DropdownMenuSeparator />
                </>
              )}
              <DropdownMenuItem
                className="text-destructive focus:bg-destructive/10 focus:text-destructive"
                onClick={() => setRevokeId(row.id)}
              >
                <Trash2 className="mr-2 size-4" /> Revoke
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        )}
      />

      <Dialog open={addOpen} onOpenChange={setAddOpen}>
        <DialogTrigger asChild>
          <Button variant="outline" size="sm">
            <Plus className="mr-2 h-4 w-4" /> Add Credential
          </Button>
        </DialogTrigger>
        <DialogContent>
          <form onSubmit={handleAddCredential}>
            <DialogHeader>
              <DialogTitle>Add Credential</DialogTitle>
              <DialogDescription>
                Create a new API credential for a provider account. The secret will be validated and never displayed again.
              </DialogDescription>
            </DialogHeader>
            <div className="grid gap-4 py-4">
              <div className="grid gap-2">
                <Label htmlFor="addAccountId">Provider Account</Label>
                <Select value={addAccountId} onValueChange={setAddAccountId}>
                  <SelectTrigger>
                    <SelectValue placeholder="Select an account" />
                  </SelectTrigger>
                  <SelectContent>
                    {accounts.length === 0 ? (
                      <SelectItem value="none" disabled>No accounts available</SelectItem>
                    ) : (
                      accounts.map((a) => (
                        <SelectItem key={a.id} value={a.id}>
                          {a.name} ({a.provider_kind})
                        </SelectItem>
                      ))
                    )}
                  </SelectContent>
                </Select>
              </div>
              <div className="grid gap-2">
                <Label htmlFor="addName">Display Name</Label>
                <Input id="addName" value={addName} onChange={(e) => setAddName(e.target.value)} placeholder="e.g. Production Key" required />
              </div>
              <div className="grid gap-2">
                <Label htmlFor="addKind">Kind</Label>
                <Select value={addKind} onValueChange={setAddKind}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="openai">OpenAI</SelectItem>
                    <SelectItem value="anthropic">Anthropic</SelectItem>
                    <SelectItem value="google">Google</SelectItem>
                    <SelectItem value="cohere">Cohere</SelectItem>
                    <SelectItem value="mistral">Mistral</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="grid gap-2">
                <Label htmlFor="addSecret">Secret (API Key)</Label>
                <Input
                  id="addSecret"
                  type="password"
                  value={addSecret}
                  onChange={(e) => setAddSecret(e.target.value)}
                  placeholder="sk-... or equivalent"
                  required
                  autoComplete="off"
                />
                <p className="text-xs text-muted-foreground">The secret will be validated before saving and never displayed again.</p>
              </div>
            </div>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setAddOpen(false)} disabled={isUpdating}>
                Cancel
              </Button>
              <Button type="submit" disabled={isUpdating || !addAccountId || !addName || !addSecret}>
                Add Credential
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <Dialog open={!!rotateId} onOpenChange={(open) => !open && setRotateId(null)}>
        <DialogContent>
          <form onSubmit={handleRotateCredential}>
            <DialogHeader>
              <DialogTitle>Rotate Credential</DialogTitle>
              <DialogDescription>
                Replace the secret for this credential. The old secret will be invalidated immediately.
              </DialogDescription>
            </DialogHeader>
            <div className="grid gap-4 py-4">
              <div className="grid gap-2">
                <Label htmlFor="rotateSecret">New Secret (API Key)</Label>
                <Input
                  id="rotateSecret"
                  type="password"
                  value={rotateSecret}
                  onChange={(e) => setRotateSecret(e.target.value)}
                  placeholder="sk-... or equivalent"
                  required
                  autoComplete="off"
                />
                <p className="text-xs text-muted-foreground">The new secret will be validated before saving and never displayed again.</p>
              </div>
            </div>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setRotateId(null)} disabled={isUpdating}>
                Cancel
              </Button>
              <Button type="submit" disabled={isUpdating || !rotateSecret}>
                Rotate
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <Dialog open={!!revokeId} onOpenChange={(open) => !open && setRevokeId(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Revoke Credential</DialogTitle>
            <DialogDescription>
              Are you sure you want to revoke this credential? It will no longer be usable for routing.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setRevokeId(null)} disabled={isUpdating}>
              Cancel
            </Button>
            <Button variant="destructive" onClick={handleRevokeCredential} disabled={isUpdating}>
              Revoke
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}