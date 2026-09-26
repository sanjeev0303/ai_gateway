import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Settings",
};

export default function SettingsPage() {
  return (
    <section>
      <h1>Settings</h1>
      <p>Workspace and organization settings arrive with the tenant milestone.</p>
    </section>
  );
}