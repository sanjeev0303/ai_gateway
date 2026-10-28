"use client";

import * as React from "react";
import { Search, Keyboard, ArrowUp, ArrowDown, ArrowRight, ExternalLink, Loader2 } from "lucide-react";
import { cn } from "cn";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Dialog,
  DialogContent,
  DialogPortal,
  DialogOverlay,
} from "@/components/ui/dialog";
import { useSession } from "@/lib/session-context";
import { useApiClient } from "@/hooks/use-api-client";
import type { SearchOptions, SearchResultItem } from "@ai-governance/api-client";

const TYPE_ICONS: Record<SearchResultItem["type"], React.ReactNode> = {
  request: <Search className="size-4" />,
  user: <Keyboard className="size-4" />,
  team: <Keyboard className="size-4" />,
  project: <Keyboard className="size-4" />,
  policy: <Keyboard className="size-4" />,
  model: <Keyboard className="size-4" />,
  provider: <Keyboard className="size-4" />,
  conversation: <Keyboard className="size-4" />,
  gateway_credential: <Keyboard className="size-4" />,
};

const TYPE_LABELS: Record<SearchResultItem["type"], string> = {
  request: "Request",
  user: "User",
  team: "Team",
  project: "Project",
  policy: "Policy",
  model: "Model",
  provider: "Provider",
  conversation: "Conversation",
  gateway_credential: "Gateway Credential",
};

interface CommandPaletteProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

function getInitials(name: string): string {
  return name
    .split(" ")
    .map((n) => n[0])
    .join("")
    .toUpperCase()
    .slice(0, 2);
}

export function CommandPalette({ open, onOpenChange }: CommandPaletteProps) {
  const [query, setQuery] = React.useState("");
  const [results, setResults] = React.useState<SearchResultItem[]>([]);
  const [selectedIndex, setSelectedIndex] = React.useState(0);
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);
  const session = useSession();
  const apiClient = useApiClient();

  const searchDebounce = React.useRef<NodeJS.Timeout | null>(null);

  const performSearch = React.useCallback(
    async (q: string) => {
      if (!q.trim()) {
        setResults([]);
        return;
      }

      if (session.status !== "authenticated") return;

      setLoading(true);
      setError(null);

      try {
        const response = await apiClient.search.search({
          q,
          limit: 15,
          types: [
            "user",
            "team",
            "project",
            "policy",
            "model",
            "provider",
            "conversation",
            "gateway_credential",
          ],
        });
        setResults(response.results);
        setSelectedIndex(0);
      } catch (err) {
        console.error("Search failed:", err);
        setError("Search failed. Please try again.");
        setResults([]);
      } finally {
        setLoading(false);
      }
    },
    [session.status]
  );

  React.useEffect(() => {
    if (searchDebounce.current) {
      clearTimeout(searchDebounce.current);
    }
    searchDebounce.current = setTimeout(() => {
      performSearch(query);
    }, 150);

    return () => {
      if (searchDebounce.current) {
        clearTimeout(searchDebounce.current);
      }
    };
  }, [query, performSearch]);

  React.useEffect(() => {
    if (open) {
      setSelectedIndex(0);
    }
  }, [open]);

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "ArrowDown") {
      e.preventDefault();
      setSelectedIndex((prev) => Math.min(prev + 1, results.length - 1));
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      setSelectedIndex((prev) => Math.max(prev - 1, 0));
    } else if (e.key === "Enter" && results[selectedIndex]) {
      e.preventDefault();
      window.location.href = results[selectedIndex].url;
      onOpenChange(false);
    } else if (e.key === "Escape") {
      onOpenChange(false);
      setQuery("");
      setResults([]);
    }
  };

  const handleResultClick = (url: string) => {
    window.location.href = url;
    onOpenChange(false);
    setQuery("");
    setResults([]);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogPortal>
        <DialogOverlay className="bg-black/30 backdrop-blur-sm" />
        <DialogContent
          className="fixed top-[20%] left-1/2 -translate-x-1/2 w-full max-w-2xl max-h-[70vh] p-0 shadow-xl"
          showCloseButton={false}
        >
          <div className="flex flex-col">
            <div className="relative p-4 border-b">
              <Search className="absolute left-4 top-1/2 -translate-y-1/2 size-5 text-muted-foreground" aria-hidden />
              <Keyboard className="absolute right-4 top-1/2 -translate-y-1/2 size-4 text-muted-foreground/50" aria-hidden />
              <Input
                type="search"
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                onKeyDown={handleKeyDown}
                placeholder="Search requests, users, policies, projects..."
                className="pl-10 pr-12 h-11 text-base"
                autoFocus
                disabled={loading}
              />
            </div>

            <div className="flex-1 overflow-hidden">
              {error && (
                <div className="p-4 text-center text-sm text-destructive bg-destructive/10 border-b">
                  {error}
                </div>
              )}

              <ScrollArea className="flex-1">
                <div className="p-2 space-y-1">
                  {loading && results.length === 0 ? (
                    Array.from({ length: 5 }).map((_, i) => (
                      <div key={i} className="flex items-center gap-3 p-3 rounded-lg hover:bg-accent">
                        <Skeleton className="h-5 w-10" />
                        <Skeleton className="flex-1 h-4 w-3/4" />
                      </div>
                    ))
                  ) : results.length === 0 && query ? (
                    <div className="flex flex-col items-center justify-center py-12 text-sm text-muted-foreground">
                      <Search className="size-8 mb-2 opacity-50" />
                      <p>No results for "{query}"</p>
                    </div>
                  ) : results.length === 0 && !query ? (
                    <div className="flex flex-col items-center justify-center py-12 text-sm text-muted-foreground">
                      <Search className="size-8 mb-2 opacity-50" />
                      <p>Start typing to search...</p>
                      <p className="text-xs mt-1">Press <kbd className="px-1.5 py-0.5 bg-muted rounded text-xs font-mono">⌘K</kbd> to open</p>
                    </div>
                  ) : (
                    results.map((result, index) => (
                      <button
                        key={`${result.type}-${result.id}`}
                        onClick={() => handleResultClick(result.url)}
                        className={cn(
                          "w-full flex items-center gap-3 p-3 rounded-lg text-left transition-colors",
                          index === selectedIndex
                            ? "bg-accent text-accent-foreground"
                            : "hover:bg-accent text-foreground"
                        )}
                        tabIndex={-1}
                      >
                        <div
                          className={cn(
                            "flex size-9 items-center justify-center rounded-lg shrink-0",
                            index === selectedIndex
                              ? "bg-accent/50"
                              : "bg-muted"
                          )}
                        >
                          {TYPE_ICONS[result.type]}
                        </div>
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2">
                            <span className="font-medium truncate">{result.name}</span>
                            <span className="text-xs px-2 py-0.5 rounded bg-muted/50 text-muted-foreground">
                              {TYPE_LABELS[result.type]}
                            </span>
                          </div>
                          {result.description && (
                            <p className="text-sm truncate text-muted-foreground">
                              {result.description}
                            </p>
                          )}
                        </div>
                        <ExternalLink className="size-4 opacity-50" aria-hidden />
                      </button>
                    ))
                  )}
                </div>
              </ScrollArea>
            </div>
          </div>
        </DialogContent>
      </DialogPortal>
    </Dialog>
  );
}

let commandPaletteOpen = false;
let commandPaletteSetOpen: ((open: boolean) => void) | null = null;

export function useCommandPalette() {
  const [open, setOpen] = React.useState(false);

  React.useEffect(() => {
    commandPaletteSetOpen = setOpen;
    return () => {
      commandPaletteSetOpen = null;
    };
  }, []);

  React.useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === "k") {
        if (
          e.target instanceof HTMLInputElement ||
          e.target instanceof HTMLTextAreaElement ||
          e.target instanceof HTMLSelectElement ||
          (e.target as HTMLElement).isContentEditable
        ) {
          return;
        }
        e.preventDefault();
        if (commandPaletteSetOpen) {
          commandPaletteSetOpen(!commandPaletteOpen);
          commandPaletteOpen = !commandPaletteOpen;
        }
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, []);

  return { open, setOpen };
}

export function CommandPaletteTrigger() {
  const { open, setOpen } = useCommandPalette();

  return (
    <CommandPalette open={open} onOpenChange={setOpen} />
  );
}