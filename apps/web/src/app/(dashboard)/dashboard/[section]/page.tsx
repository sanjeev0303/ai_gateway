"use client";

import { use } from "react";

import { EmptyState, ErrorState, PageHeader } from "@/components/ui";
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
      <div>
        <PageHeader title="Not found" />
        <EmptyState title={`No dashboard section named ${slug}`} />
      </div>
    );
  }

  const role = session.data?.claims.role;

  if (role !== undefined && !canRead(role, section.requires)) {
    return (
      <div>
        <PageHeader title={section.title} />
        <ErrorState title="Permission Denied" error={`Your role does not include ${section.requires}, so this section is not shown.`} />
      </div>
    );
  }

  const Body = section.Body;

  return (
    <div>
      <PageHeader title={section.title} description={section.blurb} />
      <div className="card">
        <Body />
      </div>
    </div>
  );
}
