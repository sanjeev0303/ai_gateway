"use client";

import * as React from "react";
import { MoreHorizontal, UserCheck, UserX } from "lucide-react";

import { Button } from "@ai-governance/console-ui/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@ai-governance/console-ui/ui/dropdown-menu";
import { ReasonDialog } from "../_components/reason-dialog";
import { disableUserAction, enableUserAction } from "../actions";

export interface UserActionRow {
  id: string;
  name: string;
  disabled: boolean;
  isSelf: boolean;
}

/** Row menu on the platform user list: disable or re-enable one person. */
export function UserActionsCell({ row }: { row: UserActionRow }) {
  const [open, setOpen] = React.useState(false);

  // The control API refuses self-disable too; hiding it avoids a dead end.
  if (row.isSelf) return null;

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="ghost" size="icon" className="size-8" aria-label={`Actions for ${row.name}`}>
            <MoreHorizontal aria-hidden />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-48">
          <DropdownMenuGroup>
            {row.disabled ? (
              <DropdownMenuItem onSelect={() => setOpen(true)}>
                <UserCheck aria-hidden />
                Re-enable user
              </DropdownMenuItem>
            ) : (
              <DropdownMenuItem variant="destructive" onSelect={() => setOpen(true)}>
                <UserX aria-hidden />
                Disable user
              </DropdownMenuItem>
            )}
          </DropdownMenuGroup>
        </DropdownMenuContent>
      </DropdownMenu>

      {row.disabled ? (
        <ReasonDialog
          open={open}
          onOpenChange={setOpen}
          title={`Re-enable ${row.name}?`}
          description="They can sign in again and their own gateway keys resolve again. Sessions ended by the disable stay ended."
          confirmLabel="Re-enable user"
          successMessage={`${row.name} can sign in again.`}
          onConfirm={(reason) => enableUserAction(row.id, reason)}
        />
      ) : (
        <ReasonDialog
          open={open}
          onOpenChange={setOpen}
          title={`Disable ${row.name}?`}
          description="They are signed out of every organization, cannot sign in, and gateway keys issued to them stop working. Their organizations and data stay as they are."
          confirmLabel="Disable user"
          destructive
          successMessage={`${row.name} is disabled.`}
          onConfirm={(reason) => disableUserAction(row.id, reason)}
        />
      )}
    </>
  );
}
