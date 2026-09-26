import type { ReactNode } from "react";

import { DocsSidebar } from "./docs-nav";

import "./docs.css";

export default function DocsLayout({ children }: { readonly children: ReactNode }) {
  return (
    <div className="docs-wrap">
      <div className="l-wrap docs-inner">
        <DocsSidebar />
        <article className="docs-main">{children}</article>
      </div>
    </div>
  );
}