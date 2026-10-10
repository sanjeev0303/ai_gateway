"use client";

import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuLabel, DropdownMenuSeparator, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { Button } from "@/components/ui/button";
import { MoreHorizontal, MailPlus } from "lucide-react";
import type { InviteRow } from "./invitations-client";

interface InviteActionsCellProps {
  row: InviteRow;
  isAdmin: boolean;
  onReissue: (email: string, role: string, id: string) => void;
  onRevoke: (id: string) => void;
  isUpdating: boolean;
}

export function InviteActionsCell({ row, isAdmin, onReissue, onRevoke, isUpdating }: InviteActionsCellProps) {
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
        {row.state === "expired" && (
          <DropdownMenuItem onClick={() => onReissue(row.email, row.role, row.id)}>
            Reissue invitation
          </DropdownMenuItem>
        )}
        {row.state === "pending" && (
          <DropdownMenuItem onClick={() => onReissue(row.email, row.role, row.id)}>
            Resend / Reissue
          </DropdownMenuItem>
        )}
        <DropdownMenuSeparator />
        <DropdownMenuItem
          className="text-destructive focus:bg-destructive/10 focus:text-destructive"
          onClick={() => onRevoke(row.id)}
        >
          Revoke
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}