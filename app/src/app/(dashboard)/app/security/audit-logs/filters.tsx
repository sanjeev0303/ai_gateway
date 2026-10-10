"use client";

import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { useCallback, useState } from "react";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";

export function AuditFilters() {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();

  const [actorId, setActorId] = useState(searchParams.get("actor_id") ?? "");
  const [action, setAction] = useState(searchParams.get("action") ?? "");
  const [resource, setResource] = useState(searchParams.get("resource_type") ?? "");

  const apply = useCallback(() => {
    const next = new URLSearchParams(searchParams.toString());
    if (actorId) next.set("actor_id", actorId);
    else next.delete("actor_id");

    if (action) next.set("action", action);
    else next.delete("action");

    if (resource) next.set("resource_type", resource);
    else next.delete("resource_type");

    router.push(`${pathname}?${next.toString()}`);
  }, [searchParams, actorId, action, resource, pathname, router]);

  const clear = useCallback(() => {
    setActorId("");
    setAction("");
    setResource("");
    router.push(pathname);
  }, [pathname, router]);

  return (
    <div className="flex flex-wrap items-center gap-3 mb-6 bg-card p-4 rounded-xl border shadow-sm">
      <div className="flex items-center gap-2">
        <label className="text-sm font-medium">Actor ID</label>
        <Input
          className="w-48 h-9"
          placeholder="User or service account ID"
          value={actorId}
          onChange={(e) => setActorId(e.target.value)}
        />
      </div>
      <div className="flex items-center gap-2">
        <label className="text-sm font-medium">Action</label>
        <Input
          className="w-48 h-9"
          placeholder="e.g. policy.create"
          value={action}
          onChange={(e) => setAction(e.target.value)}
        />
      </div>
      <div className="flex items-center gap-2">
        <label className="text-sm font-medium">Resource</label>
        <Input
          className="w-48 h-9"
          placeholder="e.g. policy"
          value={resource}
          onChange={(e) => setResource(e.target.value)}
        />
      </div>
      <div className="ml-auto flex items-center gap-2">
        <Button variant="outline" size="sm" onClick={clear}>Clear</Button>
        <Button size="sm" onClick={apply}>Apply</Button>
      </div>
    </div>
  );
}
