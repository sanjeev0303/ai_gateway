"use client";

import { use } from "react";

import { EmptyNote } from "@/components/ui";
import { canRead } from "@/lib/control-api";
import { useSession } from "@/lib/queries";
import { sectionBySlug } from "@/lib/section-registry";

export default function DashboardSectionPage({
  params,
}: {
  readonly params: Promise<{ section: string }>;
}) {
  const { section: slug } = use(params);
  const session = useSession();
  const section = sectionBySlug(slug);

  if (!section) {
    return (
      <section className="dash-panel">
        <h1>Not found</h1>
        <EmptyNote>No dashboard section named {slug}.</EmptyNote>
      </section>
    );
  }

  const role = session.data?.claims.role;

  if (role !== undefined && !canRead(role, section.requires)) {
    return (
      <section className="dash-panel">
        <h1>{section.title}</h1>
        <EmptyNote>
          Your role does not include {section.requires}, so this section is not shown.
        </EmptyNote>
      </section>
    );
  }

  const Body = section.Body;

  return (
    <>
      <header className="dash-head">
        <h1>{section.title}</h1>
        <p className="dash-note">{section.blurb}</p>
      </header>
      <section className="dash-panel">
        <Body />
      </section>
    </>
  );
}
