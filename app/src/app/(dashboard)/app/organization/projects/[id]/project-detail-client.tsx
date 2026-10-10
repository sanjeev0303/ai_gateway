"use client";

import * as React from "react";
import { MoreHorizontal, Plus, Settings, Archive, RotateCcw } from "lucide-react";
import { DataTable, type Column } from "@/components/dashboard";
import { ProjectMemberActionsCell } from "./project-member-actions-cell";
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
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { toast } from "sonner";
import { createApiClient } from "@ai-governance/api-client";
import { CONTROL_API_BASE } from "@/lib/control-api";
import { useRouter } from "next/navigation";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { formatDate } from "@/lib/format";
import { Badge } from "@/components/ui/badge";

export interface MemberRow {
  id: string;
  name: string;
  email: string;
  role: string;
  since: string;
  sinceTs: number;
}

const COLUMNS: readonly Column<MemberRow>[] = [
  { key: "name", header: "Member", primary: "name", secondary: "email", sort: "name" },
  { key: "role", header: "Role", primary: "role" },
  { key: "since", header: "Member since", primary: "since", sort: "sinceTs" },
];

export function ProjectDetailClient({
  project,
  members,
  orgMembers,
}: {
  project: any;
  members: any[];
  orgMembers: any[];
}) {
  const router = useRouter();
  const api = React.useMemo(() => createApiClient({ baseUrl: CONTROL_API_BASE }), []);

  const [rows, setRows] = React.useState<MemberRow[]>([]);
  
  React.useEffect(() => {
    setRows(
      members.map((m) => ({
        id: m.user.id,
        name: m.user.display_name || m.user.email,
        email: m.user.email,
        role: m.role,
        since: formatDate(m.member_since),
        sinceTs: Date.parse(m.member_since),
      }))
    );
  }, [members]);

  const [isUpdating, setIsUpdating] = React.useState(false);

  // Edit Project
  const [editOpen, setEditOpen] = React.useState(false);
  const [editName, setEditName] = React.useState(project.name);
  const [editDescription, setEditDescription] = React.useState(project.description);

  const handleEditProject = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      setIsUpdating(true);
      await api.projects.update(project.id, { name: editName, description: editDescription });
      toast.success("Project updated");
      setEditOpen(false);
      router.refresh();
    } catch {
      toast.error("Failed to update project");
    } finally {
      setIsUpdating(false);
    }
  };

  // Archive/Unarchive Project
  const handleToggleArchive = async (newStatus: "active" | "archived") => {
    try {
      setIsUpdating(true);
      await api.projects.setStatus(project.id, { status: newStatus });
      toast.success(newStatus === "archived" ? "Project archived" : "Project unarchived");
      router.refresh();
    } catch {
      toast.error(`Failed to ${newStatus === "archived" ? "archive" : "unarchive"} project`);
    } finally {
      setIsUpdating(false);
    }
  };

  // Add Member
  const [addOpen, setAddOpen] = React.useState(false);
  const [addUserId, setAddUserId] = React.useState("");
  const [addRole, setAddRole] = React.useState("member");

  const availableOrgMembers = orgMembers.filter((om) => !rows.find((r) => r.id === om.user.id));

  const handleAddMember = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      setIsUpdating(true);
      await api.projects.addMember(project.id, { user_id: addUserId, role: addRole });
      toast.success("Member added");
      setAddOpen(false);
      router.refresh();
    } catch {
      toast.error("Failed to add member");
    } finally {
      setIsUpdating(false);
    }
  };

  // Remove Member
  const [removeId, setRemoveId] = React.useState<string | null>(null);

  const handleRemoveMember = async () => {
    if (!removeId) return;
    try {
      setIsUpdating(true);
      await api.projects.removeMember(project.id, removeId);
      toast.success("Member removed");
      setRemoveId(null);
      router.refresh();
    } catch {
      toast.error("Failed to remove member");
    } finally {
      setIsUpdating(false);
    }
  };

  function ProjectMemberActionsCellWrapper({ row }: { row: MemberRow }) {
    return <ProjectMemberActionsCell row={row} isUpdating={isUpdating} onRemove={setRemoveId} />;
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap justify-between gap-2 items-center">
        <div className="flex gap-2">
          <Badge variant={project.status === "active" ? "default" : "secondary"}>
            {project.status}
          </Badge>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" onClick={() => setEditOpen(true)}>
            <Settings className="mr-2 size-4" /> Edit Project
          </Button>
          <Button onClick={() => setAddOpen(true)}>
            <Plus className="mr-2 size-4" /> Add Member
          </Button>
          {project.status === "active" ? (
            <Button variant="secondary" onClick={() => handleToggleArchive("archived")} disabled={isUpdating}>
              <Archive className="mr-2 size-4" /> Archive
            </Button>
          ) : (
            <Button variant="outline" onClick={() => handleToggleArchive("active")} disabled={isUpdating}>
              <RotateCcw className="mr-2 size-4" /> Unarchive
            </Button>
          )}
        </div>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Project Members</CardTitle>
        </CardHeader>
        <CardContent>
          <DataTable
            columns={COLUMNS}
            rows={rows}
            rowKey="id"
            emptyMessage="No members in this project."
            ActionsComponent={ProjectMemberActionsCellWrapper}
          />
        </CardContent>
      </Card>

      <Dialog open={editOpen} onOpenChange={setEditOpen}>
        <DialogContent>
          <form onSubmit={handleEditProject}>
            <DialogHeader>
              <DialogTitle>Edit Project</DialogTitle>
            </DialogHeader>
            <div className="grid gap-4 py-4">
              <div className="grid gap-2">
                <Label htmlFor="editName">Name</Label>
                <Input id="editName" value={editName} onChange={(e) => setEditName(e.target.value)} required />
              </div>
              <div className="grid gap-2">
                <Label htmlFor="editDescription">Description</Label>
                <Input id="editDescription" value={editDescription} onChange={(e) => setEditDescription(e.target.value)} />
              </div>
            </div>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setEditOpen(false)}>Cancel</Button>
              <Button type="submit" disabled={isUpdating}>Save Changes</Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <Dialog open={addOpen} onOpenChange={setAddOpen}>
        <DialogContent>
          <form onSubmit={handleAddMember}>
            <DialogHeader>
              <DialogTitle>Add Member</DialogTitle>
            </DialogHeader>
            <div className="grid gap-4 py-4">
              <div className="grid gap-2">
                <Label>Member</Label>
                <Select value={addUserId} onValueChange={setAddUserId}>
                  <SelectTrigger>
                    <SelectValue placeholder="Select a member..." />
                  </SelectTrigger>
                  <SelectContent>
                    {availableOrgMembers.length === 0 ? (
                      <SelectItem value="none" disabled>No members available</SelectItem>
                    ) : (
                      availableOrgMembers.map((om) => (
                        <SelectItem key={om.user.id} value={om.user.id}>
                          {om.user.display_name || om.user.email}
                        </SelectItem>
                      ))
                    )}
                  </SelectContent>
                </Select>
              </div>
              <div className="grid gap-2">
                <Label>Role</Label>
                <Select value={addRole} onValueChange={setAddRole}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="member">Member</SelectItem>
                    <SelectItem value="admin">Admin</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setAddOpen(false)}>Cancel</Button>
              <Button type="submit" disabled={isUpdating || !addUserId || addUserId === "none"}>Add Member</Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <Dialog open={!!removeId} onOpenChange={(open) => !open && setRemoveId(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Remove member</DialogTitle>
            <DialogDescription>
              Are you sure you want to remove this member from the project?
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setRemoveId(null)} disabled={isUpdating}>
              Cancel
            </Button>
            <Button variant="destructive" onClick={handleRemoveMember} disabled={isUpdating}>
              Remove
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
