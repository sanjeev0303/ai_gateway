"use client";

import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Button } from "@/components/ui/button";
import { MoreHorizontal, ShieldCheck, UserMinus } from "lucide-react";
import type { MemberRow } from "./members-client";

interface MemberActionsCellProps {
  row: MemberRow;
  isAdmin: boolean;
  currentUserRole: string;
  currentUserEmail: string;
  onRoleChange: (id: string, newRole: string) => void;
  onRemove: (id: string) => void;
  isUpdating: boolean;
}

const AVAILABLE_ROLES = [
  { value: "admin", label: "Admin" },
  { value: "member", label: "Member" },
  { value: "auditor", label: "Auditor" },
  { value: "billing_admin", label: "Billing Admin" },
] as const;

export function MemberActionsCell({
  row,
  isAdmin,
  currentUserRole,
  currentUserEmail,
  onRoleChange,
  onRemove,
  isUpdating,
}: MemberActionsCellProps) {
  if (!isAdmin) return null;

  // Cannot manage self through table actions
  if (row.email === currentUserEmail) return null;

  // Non-owners cannot modify or remove owners
  const isTargetOwner = row.role === "owner";
  const isCallerOwner = currentUserRole === "owner";
  if (isTargetOwner && !isCallerOwner) return null;

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="icon" className="size-8 p-0" disabled={isUpdating}>
          <span className="sr-only">Open menu</span>
          <MoreHorizontal className="size-4" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-48">
        <DropdownMenuLabel>Member Actions</DropdownMenuLabel>
        <DropdownMenuSub>
          <DropdownMenuSubTrigger>
            <ShieldCheck className="mr-2 size-4 text-muted-foreground" />
            Change Role
          </DropdownMenuSubTrigger>
          <DropdownMenuSubContent>
            {AVAILABLE_ROLES.map((role) => (
              <DropdownMenuItem
                key={role.value}
                disabled={row.role === role.value || (!isCallerOwner && role.value === "admin")}
                onClick={() => onRoleChange(row.id, role.value)}
              >
                {role.label} {row.role === role.value ? "✓" : ""}
              </DropdownMenuItem>
            ))}
          </DropdownMenuSubContent>
        </DropdownMenuSub>
        <DropdownMenuSeparator />
        <DropdownMenuItem
          className="text-destructive focus:bg-destructive/10 focus:text-destructive"
          onClick={() => onRemove(row.id)}
        >
          <UserMinus className="mr-2 size-4" />
          Remove member
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}