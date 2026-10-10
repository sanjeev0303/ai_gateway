"use client";

import * as React from "react";
import { MoreHorizontal } from "lucide-react";
import { DataTable, type Column } from "@/components/dashboard";
import { MemberActionsCell } from "./member-actions-cell";
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
} from "@/components/ui/dialog";
import { toast } from "sonner";
import { createApiClient } from "@ai-governance/api-client";
import { CONTROL_API_BASE } from "@/lib/control-api";
import { useSession } from "@/lib/session-context";
import { useRouter } from "next/navigation";

export interface MemberRow {
  id: string;
  name: string;
  email: string;
  role: string;
  status: string;
  spend: string;
}

const COLUMNS: readonly Column<MemberRow>[] = [
  { key: "name", header: "Member", primary: "name", secondary: "email", sort: "name" },
  { key: "role", header: "Role", primary: "role" },
  { key: "status", header: "Status", primary: "status" },
  { key: "spend", header: "Spend (30d)", primary: "spend", format: "usd", align: "right", sort: "spend" },
];

export function MembersClient({ initialRows }: { initialRows: MemberRow[] }) {
  const router = useRouter();
  const session = useSession();
  const isAdmin = session.status === "authenticated" && session.session.user.role === "admin";
  const currentUserEmail = session.status === "authenticated" ? session.session.user.email : "";

  const [rows, setRows] = React.useState(initialRows);
  const [removeId, setRemoveId] = React.useState<string | null>(null);
  const [isUpdating, setIsUpdating] = React.useState(false);

  const api = React.useMemo(() => createApiClient({ baseUrl: CONTROL_API_BASE }), []);

  const handleRoleChange = async (id: string, newRole: string) => {
    try {
      setIsUpdating(true);
      await api.members.setRole(id, { role: newRole });
      setRows((prev) => prev.map((r) => (r.id === id ? { ...r, role: newRole } : r)));
      toast.success("Role updated successfully");
      router.refresh();
    } catch (error) {
      toast.error("Failed to update role");
    } finally {
      setIsUpdating(false);
    }
  };

  const handleRemove = async () => {
    if (!removeId) return;
    try {
      setIsUpdating(true);
      await api.members.remove(removeId);
      setRows((prev) => prev.filter((r) => r.id !== removeId));
      toast.success("Member removed successfully");
      router.refresh();
    } catch (error) {
      toast.error("Failed to remove member");
    } finally {
      setIsUpdating(false);
      setRemoveId(null);
    }
  };

  function MemberActionsCellWrapper({ row }: { row: MemberRow }) {
    return <MemberActionsCell row={row} isAdmin={isAdmin} currentUserEmail={currentUserEmail} onRoleChange={handleRoleChange} onRemove={handleRemove} isUpdating={isUpdating} />;
  }

  return (
    <>
      <DataTable
        columns={COLUMNS}
        rows={rows}
        rowKey="id"
        emptyMessage="No members in this organization."
        ActionsComponent={MemberActionsCellWrapper}
      />

      <Dialog open={!!removeId} onOpenChange={(open) => !open && setRemoveId(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Remove member</DialogTitle>
            <DialogDescription>
              Are you sure you want to remove this member? They will lose access to the organization immediately.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setRemoveId(null)} disabled={isUpdating}>
              Cancel
            </Button>
            <Button variant="destructive" onClick={handleRemove} disabled={isUpdating}>
              Remove
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
