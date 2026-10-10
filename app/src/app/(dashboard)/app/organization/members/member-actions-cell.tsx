"use client";

import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuLabel, DropdownMenuSeparator, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { Button } from "@/components/ui/button";
import { MoreHorizontal, UserPlus } from "lucide-react";
import type { MemberRow } from "./members-client";

interface MemberActionsCellProps {
  row: MemberRow;
  isAdmin: boolean;
  currentUserEmail: string;
  onRoleChange: (id: string, newRole: string) => void;
  onRemove: (id: string) => void;
  isUpdating: boolean;
}

export function MemberActionsCell({ row, isAdmin, currentUserEmail, onRoleChange, onRemove, isUpdating }: MemberActionsCellProps) {
  if (!isAdmin) return null;

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="icon" className="size-8 p-0" disabled={isUpdating}>
          <span className="sr-only">Open menu</span>
          <MoreHorizontal className="size-4" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuLabel>Actions</DropdownMenuLabel>
        {row.email !== currentUserEmail && (
          <>
            <DropdownMenuItem onClick={() => onRoleChange(row.id, row.role === "admin" ? "member" : "admin")}>
              Make {row.role === "admin" ? "Member" : "Admin"}
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem
              className="text-destructive focus:bg-destructive/10 focus:text-destructive"
              onClick={() => onRemove(row.id)}
            >
              Remove member
            </DropdownMenuItem>
          </>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}