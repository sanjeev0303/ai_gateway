"use client";

import * as React from "react";
import { Check, Copy } from "lucide-react";

import { Button } from "@/components/ui/button";
import { cn } from "cn";

const LANGUAGE_LABELS: Record<string, string> = {
  bash: "cURL",
  typescript: "TypeScript",
  python: "Python",
};

export interface CodeBlockProps {
  code: string;
  /** Short caption above the block, e.g. the endpoint being called. */
  label?: string;
  language?: string;
  className?: string;
}

/**
 * Minimal code surface. No syntax-highlighting dependency: the snippets here are
 * short and a monospace block with a working copy button is the whole job.
 */
export function CodeBlock({
  code,
  label,
  language,
  className,
}: CodeBlockProps) {
  const [copied, setCopied] = React.useState(false);
  const resetTimer = React.useRef<ReturnType<typeof setTimeout>>(undefined);

  React.useEffect(() => () => clearTimeout(resetTimer.current), []);

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(code);
      setCopied(true);
      clearTimeout(resetTimer.current);
      resetTimer.current = setTimeout(() => setCopied(false), 1600);
    } catch {
      // Clipboard access can be denied; leaving the code selectable is enough.
    }
  };

  return (
    <figure
      className={cn(
        "overflow-hidden rounded-lg border bg-muted/40 dark:bg-input/20",
        className,
      )}
    >
      <figcaption className="flex items-center justify-between gap-3 border-b px-3 py-1.5">
        <span className="flex min-w-0 items-center gap-2 text-xs text-muted-foreground">
          {label && <span className="truncate font-mono">{label}</span>}
          {language && (
            <span className="rounded bg-muted px-1.5 py-0.5 font-medium text-foreground/80">
              {LANGUAGE_LABELS[language] ?? language}
            </span>
          )}
        </span>
        <Button
          type="button"
          size="xs"
          variant="ghost"
          onClick={copy}
          className="text-muted-foreground"
        >
          {copied ? (
            <>
              <Check aria-hidden /> Copied
            </>
          ) : (
            <>
              <Copy aria-hidden /> Copy
            </>
          )}
        </Button>
      </figcaption>
      <pre className="overflow-x-auto p-3 text-xs leading-relaxed">
        <code className="font-mono">{code}</code>
      </pre>
    </figure>
  );
}
