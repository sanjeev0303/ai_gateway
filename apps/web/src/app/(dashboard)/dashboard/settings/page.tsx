import type { Metadata } from "next";
import Link from "next/link";

import { ThemeSwitch } from "@/components/theme-switch";
import { UnavailablePanel } from "@/lib/metrics";

export const metadata: Metadata = {
  title: "Settings",
};

type Area = {
  readonly title: string;
  readonly blurb: string;
  readonly needs?: readonly string[];
  readonly href?: string;
  readonly action?: string;
};

const AREAS: readonly Area[] = [
  {
    title: "Organization",
    blurb: "Display name, slug, and organization defaults.",
    needs: ["GET /v1/organization", "PATCH /v1/organization"],
  },
  {
    title: "Members",
    blurb: "Membership is managed as part of the organization directory.",
    href: "/dashboard/members",
    action: "Manage members",
  },
  {
    title: "Teams",
    blurb: "Teams are managed as part of the organization directory.",
    href: "/dashboard/org-teams",
    action: "Manage teams",
  },
  {
    title: "Authentication",
    blurb: "Session lifetime, SSO, and password policy for the control plane.",
    needs: ["GET /v1/settings/authentication", "PATCH /v1/settings/authentication"],
  },
  {
    title: "Security",
    blurb: "IP allowlist and control-plane auth policy.",
    needs: ["GET /v1/settings/security", "PATCH /v1/settings/security"],
  },
  {
    title: "Providers",
    blurb: "Provider connections and their health checks live on the platform pages.",
    href: "/dashboard/provider-accounts",
    action: "Manage providers",
  },
  {
    title: "Privacy",
    blurb: "Privacy mode controls whether request and response content may be stored.",
    needs: ["GET /v1/settings/privacy", "PATCH /v1/settings/privacy"],
  },
  {
    title: "Conversation retention",
    blurb: "How long conversation records are kept.",
    needs: ["GET /v1/settings/retention/conversations", "PATCH /v1/settings/retention/conversations"],
  },
  {
    title: "Audit retention",
    blurb: "How long audit records are kept.",
    needs: ["GET /v1/settings/retention/audit", "PATCH /v1/settings/retention/audit"],
  },
  {
    title: "Notifications",
    blurb: "Email destinations for alerts and budget warnings.",
    needs: ["GET /v1/settings/notifications", "PATCH /v1/settings/notifications"],
  },
  {
    title: "Webhooks",
    blurb: "Outbound webhook destinations and delivery settings.",
    needs: ["GET /v1/settings/webhooks", "POST /v1/settings/webhooks"],
  },
  {
    title: "Billing",
    blurb: "Plan and invoice settings. Provider AI spend is tracked separately under Costs.",
    needs: ["GET /v1/billing/subscription", "PATCH /v1/billing/subscription"],
  },
  {
    title: "Developer",
    blurb: "Start integrating against the gateway.",
    href: "/dashboard/quick-start",
    action: "Open quick start",
  },
];

export default function SettingsPage() {
  return (
    <section>
      <h1>Settings</h1>
      <div className="card settings-card">
        <div className="settings-row">
          <div className="settings-copy">
            <h2 className="settings-row-title">Appearance</h2>
            <p className="settings-row-note">Light, dark, or follow your system setting.</p>
          </div>
          <ThemeSwitch />
        </div>
      </div>
      {AREAS.map((area) => (
        <div key={area.title} className="card settings-card">
          <div className="settings-row">
            <div className="settings-copy">
              <h2 className="settings-row-title">{area.title}</h2>
              <p className="settings-row-note">{area.blurb}</p>
            </div>
            {area.href !== undefined && (
              <Link className="settings-link" href={area.href}>
                {area.action}
              </Link>
            )}
          </div>
          {area.needs !== undefined && <UnavailablePanel what={`${area.title} settings`} needs={area.needs} />}
        </div>
      ))}
    </section>
  );
}
