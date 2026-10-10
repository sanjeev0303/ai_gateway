"use client";

import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuLabel, DropdownMenuSeparator, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { Button } from "@/components/ui/button";
import { MoreHorizontal, RotateCcw, Trash2 } from "lucide-react";
import type { CredentialRow } from "./client";

interface CredentialActionsCellProps {
  row: CredentialRow;
  onRotate: (id: string, secret: string) => void;
  onRevoke: (id: string) => void;
  isUpdating: boolean;
}

export function CredentialActionsCell({ row, onRotate, onRevoke, isUpdating }: CredentialActionsCellProps) {
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
        {row.status === "active" && (
          <>
            <DropdownMenuItem onClick={() => onRotate(row.id, "")}>
              <RotateCcw className="mr-2 size-4" /> Rotate
            </DropdownMenuItem>
            <DropdownMenuSeparator />
          </>
        )}
        <DropdownMenuItem
          className="text-destructive focus:bg-destructive/10 focus:text-destructive"
          onClick={() => onRevoke(row.id)}
        >
          <Trash2 className="mr-2 size-4" /> Revoke
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}