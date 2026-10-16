"use client";

import * as React from "react";
import { Plus } from "lucide-react";
import { toast } from "sonner";

import { useRouter } from "next/navigation";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { FormDialog } from "@/components/dashboard/dialogs";
import { createProviderAccount } from "./actions";

export function CreateProviderAccountButton() {
  const router = useRouter();
  const [open, setOpen] = React.useState(false);
  const [providerKind, setProviderKind] = React.useState("openai");

  return (
    <>
      <Button onClick={() => setOpen(true)} size="sm">
        <Plus className="mr-2 h-4 w-4" />
        New provider account
      </Button>

      <FormDialog
        open={open}
        onOpenChange={setOpen}
        title="Add Provider Account"
        description="Register a new provider endpoint."
        action={createProviderAccount}
        submitLabel="Add Account"
        onSuccess={() => {
          toast.success("Provider account added");
          router.refresh();
        }}
      >
        <div className="grid gap-4">
          <div className="grid gap-2">
            <Label htmlFor="provider_kind">Provider</Label>
            <input type="hidden" name="provider_kind" value={providerKind} />
            <Select value={providerKind} onValueChange={setProviderKind}>
              <SelectTrigger id="provider_kind">
                <SelectValue placeholder="Select provider" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="openai">OpenAI</SelectItem>
                <SelectItem value="anthropic">Anthropic</SelectItem>
                <SelectItem value="gemini">Google Gemini</SelectItem>
                <SelectItem value="azure_openai">Azure OpenAI</SelectItem>
                <SelectItem value="custom">Custom (OpenAI-compatible)</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="grid gap-2">
            <Label htmlFor="name">Display Name</Label>
            <Input id="name" name="name" placeholder="e.g. Acme Org OpenAI" required />
          </div>
          <div className="grid gap-2">
            <Label htmlFor="base_url">Base URL</Label>
            <Input
              id="base_url"
              name="base_url"
              placeholder="Leave blank for provider default"
            />
          </div>
        </div>
      </FormDialog>
    </>
  );
}
