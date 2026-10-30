"use client";

import { useRouter, useSearchParams, usePathname } from "next/navigation";
import { useCallback, useState } from "react";
import { Search, X } from "lucide-react";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

export function ObserveFilters() {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();

  const createQueryString = useCallback(
    (name: string, value: string) => {
      const params = new URLSearchParams(searchParams.toString());
      if (value) {
        params.set(name, value);
      } else {
        params.delete(name);
      }
      return params.toString();
    },
    [searchParams]
  );

  const setFilter = (name: string, value: string) => {
    router.push(pathname + "?" + createQueryString(name, value));
  };

  return (
    <div className="flex flex-wrap gap-4 items-center mb-6">
      <div className="flex flex-col gap-1 w-full sm:w-auto">
        <label className="text-xs text-muted-foreground font-medium">Provider</label>
        <Select
          value={searchParams.get("provider") || "all"}
          onValueChange={(val) => setFilter("provider", val === "all" ? "" : val)}
        >
          <SelectTrigger className="w-[180px]">
            <SelectValue placeholder="All Providers" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All Providers</SelectItem>
            <SelectItem value="gemini">Google Gemini</SelectItem>
            <SelectItem value="mistral">Mistral AI</SelectItem>
            <SelectItem value="huggingface">Hugging Face</SelectItem>
            <SelectItem value="openai">OpenAI</SelectItem>
            <SelectItem value="anthropic">Anthropic</SelectItem>
            <SelectItem value="azure_openai">Azure OpenAI</SelectItem>
            <SelectItem value="custom">Custom Provider</SelectItem>
          </SelectContent>
        </Select>
      </div>

      <div className="flex flex-col gap-1 w-full sm:w-auto">
        <label className="text-xs text-muted-foreground font-medium">Status</label>
        <Select
          value={searchParams.get("status") || "all"}
          onValueChange={(val) => setFilter("status", val === "all" ? "" : val)}
        >
          <SelectTrigger className="w-[180px]">
            <SelectValue placeholder="All Statuses" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All Statuses</SelectItem>
            <SelectItem value="succeeded">Succeeded</SelectItem>
            <SelectItem value="failed">Failed</SelectItem>
            <SelectItem value="cancelled">Cancelled</SelectItem>
          </SelectContent>
        </Select>
      </div>
      
      <div className="flex flex-col gap-1 w-full sm:w-auto">
        <label className="text-xs text-muted-foreground font-medium">Project ID</label>
        <Input 
          placeholder="Project ID..." 
          className="w-[180px]" 
          defaultValue={searchParams.get("project_id") || ""}
          onBlur={(e) => setFilter("project_id", e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter") {
              setFilter("project_id", e.currentTarget.value);
            }
          }}
        />
      </div>

      <div className="flex flex-col gap-1 w-full sm:w-auto">
        <label className="text-xs text-muted-foreground font-medium">User ID</label>
        <Input 
          placeholder="User ID..." 
          className="w-[180px]" 
          defaultValue={searchParams.get("user_id") || ""}
          onBlur={(e) => setFilter("user_id", e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter") {
              setFilter("user_id", e.currentTarget.value);
            }
          }}
        />
      </div>

      <div className="flex flex-col gap-1 w-full sm:w-auto">
        <label className="text-xs text-muted-foreground font-medium">Model</label>
        <Input 
          placeholder="Model name..." 
          className="w-[180px]" 
          defaultValue={searchParams.get("model") || ""}
          onBlur={(e) => setFilter("model", e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter") {
              setFilter("model", e.currentTarget.value);
            }
          }}
        />
      </div>

      <div className="flex flex-col gap-1 w-full sm:w-auto">
        <label className="text-xs text-muted-foreground font-medium">Team ID</label>
        <Input 
          placeholder="Team ID..." 
          className="w-[180px]" 
          defaultValue={searchParams.get("team_id") || ""}
          onBlur={(e) => setFilter("team_id", e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter") {
              setFilter("team_id", e.currentTarget.value);
            }
          }}
        />
      </div>

      <div className="flex flex-col gap-1 w-full sm:w-auto">
        <label className="text-xs text-muted-foreground font-medium">Service Account</label>
        <Input 
          placeholder="SA ID..." 
          className="w-[180px]" 
          defaultValue={searchParams.get("service_account") || ""}
          onBlur={(e) => setFilter("service_account", e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter") {
              setFilter("service_account", e.currentTarget.value);
            }
          }}
        />
      </div>
    </div>
  );
}
