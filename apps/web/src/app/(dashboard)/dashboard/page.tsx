import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Overview",
};

export default function DashboardOverview() {
  return (
    <section>
      <h1>Overview</h1>
      <p>
        Usage, budgets, and policy health will surface here once the control
        plane is wired.
      </p>
    </section>
  );
}