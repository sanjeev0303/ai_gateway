"use client";

import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { useCallback, useEffect, useState } from "react";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { loadDirectory } from "@/components/dashboard/directory-actions";
import { SearchSelect, type SearchSelectOption } from "@/components/dashboard/search-select";

export function AuditFilters() {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();

  const [actorId, setActorId] = useState(searchParams.get("actor_id") ?? "");
  const [action, setAction] = useState(searchParams.get("action") ?? "");
  const [resource, setResource] = useState(searchParams.get("resource_type") ?? "");

  const [actors, setActors] = useState<SearchSelectOption[] | null>(null);
  useEffect(() => {
    loadDirectory()
      .then((d) => setActors([...d.users, ...d.serviceAccounts]))
      .catch(() => setActors([]));
  }, []);

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
    <div className="grid grid-cols-1 items-end gap-3 rounded-2xl border border-border bg-card p-4 sm:grid-cols-2 lg:grid-cols-[repeat(3,minmax(0,1fr))_auto]">
      <div className="grid gap-1.5">
        <Label htmlFor="audit-actor" className="text-xs font-medium text-muted-foreground">Who</Label>
        <SearchSelect
          id="audit-actor"
          value={actorId}
          onChange={setActorId}
          options={actors ?? []}
          loading={actors === null}
          placeholder="Anyone"
          clearLabel="Anyone"
        />
      </div>
      <div className="grid gap-1.5">
        <Label htmlFor="audit-action" className="text-xs font-medium text-muted-foreground">Action</Label>
        <Input
          id="audit-action"
          placeholder="policy.create…"
          autoComplete="off"
          value={action}
          onChange={(e) => setAction(e.target.value)}
        />
      </div>
      <div className="grid gap-1.5">
        <Label htmlFor="audit-resource" className="text-xs font-medium text-muted-foreground">Resource</Label>
        <Input
          id="audit-resource"
          placeholder="policy…"
          autoComplete="off"
          value={resource}
          onChange={(e) => setResource(e.target.value)}
        />
      </div>
      <div className="flex items-center gap-2">
        <Button variant="outline" size="sm" onClick={clear}>Clear</Button>
        <Button size="sm" onClick={apply}>Apply</Button>
      </div>
    </div>
  );
}
