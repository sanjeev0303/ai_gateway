"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { Trash2 } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { confirmAction } from "@/components/dashboard";
import { deleteConversationAction } from "../actions";

export function DeleteConversationButton({ id }: { id: string }) {
  const router = useRouter();
  const [pending, setPending] = React.useState(false);

  const onDelete = async () => {
    const confirmed = await confirmAction({
      title: "Delete this conversation?",
      description: "Its messages and their text are removed for good. Usage totals already counted are not changed.",
      confirmLabel: "Delete conversation",
      destructive: true,
    });
    if (!confirmed) return;

    setPending(true);
    const result = await deleteConversationAction(id);
    if (!result.success) {
      setPending(false);
      toast.error(result.error ?? "Could not delete the conversation");
      return;
    }
    toast.success("Conversation deleted");
    router.push("/app/observe/conversations");
  };

  return (
    <Button variant="outline" onClick={onDelete} disabled={pending}>
      <Trash2 aria-hidden />
      {pending ? "Deleting…" : "Delete conversation"}
    </Button>
  );
}
