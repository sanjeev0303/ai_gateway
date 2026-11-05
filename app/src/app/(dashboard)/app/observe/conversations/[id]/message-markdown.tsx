import * as React from "react";
import ReactMarkdown, { type Components } from "react-markdown";
import remarkGfm from "remark-gfm";
import { CodeBlock } from "@/components/developer/code-block";

/**
 * Model replies are markdown. Raw HTML is never rendered (react-markdown escapes
 * it), and unsafe link protocols are dropped by its default URL filter.
 */
const components: Components = {
  p: ({ children }) => <p className="my-2 first:mt-0 last:mb-0">{children}</p>,
  strong: ({ children }) => <strong className="font-semibold">{children}</strong>,
  em: ({ children }) => <em className="italic">{children}</em>,
  h1: ({ children }) => <h3 className="mt-4 mb-2 text-base font-semibold first:mt-0">{children}</h3>,
  h2: ({ children }) => <h3 className="mt-4 mb-2 text-base font-semibold first:mt-0">{children}</h3>,
  h3: ({ children }) => <h4 className="mt-3 mb-1.5 font-semibold first:mt-0">{children}</h4>,
  h4: ({ children }) => <h4 className="mt-3 mb-1.5 font-semibold first:mt-0">{children}</h4>,
  ul: ({ children }) => <ul className="my-2 list-disc space-y-1 pl-5 marker:text-muted-foreground">{children}</ul>,
  ol: ({ children }) => <ol className="my-2 list-decimal space-y-1 pl-5 marker:text-muted-foreground">{children}</ol>,
  li: ({ children }) => <li className="pl-0.5">{children}</li>,
  blockquote: ({ children }) => (
    <blockquote className="my-2 border-l-2 border-border pl-3 text-muted-foreground">{children}</blockquote>
  ),
  hr: () => <hr className="my-3 border-border" />,
  a: ({ href, children }) => (
    <a href={href} target="_blank" rel="noopener noreferrer nofollow" className="font-medium text-primary underline underline-offset-4">
      {children}
    </a>
  ),
  table: ({ children }) => (
    <div className="my-3 overflow-x-auto rounded-lg border border-border">
      <table className="w-full text-left text-sm">{children}</table>
    </div>
  ),
  th: ({ children }) => <th className="border-b border-border bg-muted/50 px-3 py-1.5 font-semibold">{children}</th>,
  td: ({ children }) => <td className="border-b border-border px-3 py-1.5 last:border-b-0">{children}</td>,
  code: ({ children }) => (
    <code className="rounded bg-muted px-1 py-0.5 font-mono text-[0.85em]">{children}</code>
  ),
  // A fenced block: hand its text and language to the shared code block, which has a copy button.
  pre: ({ children }) => {
    const code = React.Children.only(children) as React.ReactElement<{ className?: string; children?: React.ReactNode }>;
    const language = /language-([\w+#-]+)/.exec(code.props.className ?? "")?.[1];
    return <CodeBlock code={String(code.props.children ?? "").replace(/\n$/, "")} language={language} className="my-3" />;
  },
};

export function MessageMarkdown({ text }: { text: string }) {
  return (
    <ReactMarkdown remarkPlugins={[remarkGfm]} components={components}>
      {text}
    </ReactMarkdown>
  );
}
