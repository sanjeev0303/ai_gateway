import type { Metadata } from "next";

import { ThemeSwitch } from "@/components/theme-switch";

export const metadata: Metadata = {
  title: "Settings",
};

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
      <p>Workspace and organization settings arrive with the tenant milestone.</p>
    </section>
  );
}