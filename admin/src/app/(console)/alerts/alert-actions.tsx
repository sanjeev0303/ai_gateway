"use client";

import * as React from "react";

import { Spinner } from "@ai-governance/console-ui/dashboard/state";
import { Button } from "@ai-governance/console-ui/ui/button";
import { showError, showSuccess } from "@ai-governance/console-ui/lib/toast";
import { acknowledgeAlertAction } from "../actions";

export interface AlertActionRow {
  id: string;
  open: boolean;
}

/** Acknowledge marks an alert as seen; the monitor resolves it when the problem clears. */
export function AlertActionsCell({ row }: { row: AlertActionRow }) {
  const [pending, startTransition] = React.useTransition();
  if (!row.open) return null;

  return (
    <Button
      variant="outline"
      size="sm"
      disabled={pending}
      onClick={() =>
        startTransition(async () => {
          const result = await acknowledgeAlertAction(row.id);
          if (result.error) showError(result.error);
          else showSuccess("Alert acknowledged.");
        })
      }
    >
      {pending && <Spinner />}
      Acknowledge
    </Button>
  );
}
