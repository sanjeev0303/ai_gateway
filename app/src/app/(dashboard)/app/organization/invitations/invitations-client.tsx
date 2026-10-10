"use client";

import * as React from "react";
import { Copy, MoreHorizontal, MailPlus, Check } from "lucide-react";
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
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { toast } from "sonner";
import { createApiClient } from "@ai-governance/api-client";
import { CONTROL_API_BASE } from "@/lib/control-api";
import { useSession } from "@/lib/session-context";
import { useRouter } from "next/navigation";
import { ROLE_ORDER } from "@/lib/org-tones";

export interface InviteRow {
  id: string;
  email: string;
  role: string;
  roleTone: any;
  state: string;
  stateTone: any;
  sent: string;
  sentTs: number;
  expires: string;
}

const COLUMNS: readonly Column<InviteRow>[] = [
  { key: "email", header: "Invitee", primary: "email", sort: "email" },
  { key: "role", header: "Role", primary: "role", tone: "roleTone" },
  { key: "state", header: "State", primary: "state", tone: "stateTone" },
  { key: "sent", header: "Sent", primary: "sent", align: "right", sort: "sentTs" },
  { key: "expires", header: "Expires", primary: "expires", align: "right" },
];

export function InviteDialog({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const [open, setOpen] = React.useState(false);
  const [email, setEmail] = React.useState("");
  const [role, setRole] = React.useState("member");
  const [isSubmitting, setIsSubmitting] = React.useState(false);
  const [result, setResult] = React.useState<{ token: string; email: string } | null>(null);
  const [copied, setCopied] = React.useState(false);

  const api = React.useMemo(() => createApiClient({ baseUrl: CONTROL_API_BASE }), []);

  const handleInvite = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      setIsSubmitting(true);
      const res = await api.invitations.invite({ email, role });
      setResult({ token: res.token, email });
      router.refresh();
    } catch (error) {
      toast.error("Failed to send invitation");
    } finally {
      setIsSubmitting(false);
    }
  };

  const copyToken = () => {
    if (!result) return;
    const link = `${window.location.origin}/accept-invite?token=${result.token}`;
    navigator.clipboard.writeText(link);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  if (result) {
    return (
      <Dialog open={open} onOpenChange={(v) => {
        setOpen(v);
        if (!v) {
          setResult(null);
          setEmail("");
        }
      }}>
        <DialogTrigger asChild>{children}</DialogTrigger>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Invitation sent</DialogTitle>
            <DialogDescription>
              An invitation has been created for {result.email}. Share this link with them to accept.
            </DialogDescription>
          </DialogHeader>
          <div className="flex items-center gap-2 mt-4">
            <Input readOnly value={`${window.location.origin}/accept-invite?token=${result.token}`} />
            <Button size="icon" onClick={copyToken}>
              {copied ? <Check className="size-4 text-green-500" /> : <Copy className="size-4" />}
            </Button>
          </div>
          <DialogFooter className="mt-4">
            <Button onClick={() => setOpen(false)}>Done</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    );
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>{children}</DialogTrigger>
      <DialogContent>
        <form onSubmit={handleInvite}>
          <DialogHeader>
            <DialogTitle>Invite a new member</DialogTitle>
            <DialogDescription>
              They will receive an invitation to join the organization.
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="grid gap-2">
              <Label htmlFor="email">Email</Label>
              <Input
                id="email"
                type="email"
                required
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="colleague@example.com"
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="role">Role</Label>
              <Select value={role} onValueChange={setRole}>
                <SelectTrigger>
                  <SelectValue placeholder="Select a role" />
                </SelectTrigger>
                <SelectContent>
                  {ROLE_ORDER.map((r) => (
                    <SelectItem key={r} value={r}>
                      {r.charAt(0).toUpperCase() + r.slice(1)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setOpen(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={isSubmitting || !email}>
              Invite
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

export function InvitationsClient({ initialRows, oldestStr }: { initialRows: InviteRow[], oldestStr: string | null }) {
  const router = useRouter();
  const session = useSession();
  const isAdmin = session.status === "authenticated" && session.session.user.role === "admin";

  const [rows, setRows] = React.useState(initialRows);
  const [revokeId, setRevokeId] = React.useState<string | null>(null);
  const [isUpdating, setIsUpdating] = React.useState(false);

  // Sync rows if server rows change
  React.useEffect(() => {
    setRows(initialRows);
  }, [initialRows]);

  const api = React.useMemo(() => createApiClient({ baseUrl: CONTROL_API_BASE }), []);

  const handleRevoke = async () => {
    if (!revokeId) return;
    try {
      setIsUpdating(true);
      await api.invitations.revoke(revokeId);
      setRows((prev) => prev.filter((r) => r.id !== revokeId));
      toast.success("Invitation revoked");
      router.refresh();
    } catch (error) {
      toast.error("Failed to revoke invitation");
    } finally {
      setIsUpdating(false);
      setRevokeId(null);
    }
  };

  const handleReissue = async (email: string, role: string, idToRevoke: string) => {
    try {
      setIsUpdating(true);
      await api.invitations.revoke(idToRevoke);
      await api.invitations.invite({ email, role });
      toast.success("Invitation reissued");
      router.refresh();
    } catch (error) {
      toast.error("Failed to reissue invitation");
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
        toolbar={
          oldestStr && (
            <p className="text-sm text-muted-foreground">
              Oldest invitation sent {oldestStr}.
            </p>
          )
        }
        emptyMessage="No open invitations."
        actions={isAdmin ? (row) => (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="icon" className="size-8 p-0" disabled={isUpdating}>
                <span className="sr-only">Open menu</span>
                <MoreHorizontal className="size-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuLabel>Actions</DropdownMenuLabel>
              {row.state === "expired" && (
                <DropdownMenuItem onClick={() => handleReissue(row.email, row.role, row.id)}>
                  Reissue invitation
                </DropdownMenuItem>
              )}
              {row.state === "pending" && (
                <DropdownMenuItem onClick={() => handleReissue(row.email, row.role, row.id)}>
                  Resend / Reissue
                </DropdownMenuItem>
              )}
              <DropdownMenuSeparator />
              <DropdownMenuItem
                className="text-destructive focus:bg-destructive/10 focus:text-destructive"
                onClick={() => setRevokeId(row.id)}
              >
                Revoke
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        ) : undefined}
      />

      <Dialog open={!!revokeId} onOpenChange={(open) => !open && setRevokeId(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Revoke invitation</DialogTitle>
            <DialogDescription>
              Are you sure you want to revoke this invitation? The recipient will no longer be able to accept it.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setRevokeId(null)} disabled={isUpdating}>
              Cancel
            </Button>
            <Button variant="destructive" onClick={handleRevoke} disabled={isUpdating}>
              Revoke
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
