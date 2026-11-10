"use client";

import * as React from "react";

import { Button } from "@ai-governance/console-ui/ui/button";
import { ReasonDialog } from "../_components/reason-dialog";
import { setModelActiveAction } from "../actions";

export interface ModelActionRow {
  id: string;
  name: string;
  active: boolean;
}

/** Switches one model on or off in the shared catalog, for every organization. */
export function ModelActionsCell({ row }: { row: ModelActionRow }) {
  const [open, setOpen] = React.useState(false);
  const next = !row.active;

  return (
    <>
      <Button variant={row.active ? "ghost" : "outline"} size="sm" onClick={() => setOpen(true)}>
        {row.active ? "Deactivate" : "Activate"}
      </Button>
      <ReasonDialog
        open={open}
        onOpenChange={setOpen}
        title={row.active ? `Deactivate ${row.name}?` : `Activate ${row.name}?`}
        description={
          row.active
            ? "The gateway refuses this model for every organization until it is activated again."
            : "Every organization whose policy allows it can call this model again."
        }
        confirmLabel={row.active ? "Deactivate model" : "Activate model"}
        destructive={row.active}
        successMessage={row.active ? `${row.name} deactivated.` : `${row.name} activated.`}
        onConfirm={(reason) => setModelActiveAction(row.id, next, reason)}
      />
    </>
  );
}
