import { SpecView } from "@/lib/spec";

const SEPARATION_NOTE =
  "Platform subscription spend and provider AI spend are different numbers and are never combined here. Provider AI spend — the cost of tokens your team actually consumed — lives on the Costs page under Observe.";

export function UsageAndSpendBody() {
  return (
    <SpecView
      blurb="What this organization consumes against its plan, measured in platform units rather than provider dollars."
      needs={["GET /v1/billing/usage", "GET /v1/billing/plan"]}
      groups={[
        { title: "Platform usage", items: ["seats", "requests included", "gateway calls", "overage"] },
        { title: "Plan limits", items: ["quota per unit", "period", "reset"] },
        { title: "Remaining quota", items: ["consumed", "remaining", "percent remaining"] },
      ]}
      note={SEPARATION_NOTE}
    />
  );
}

export function SubscriptionBody() {
  return (
    <SpecView
      blurb="The SaaS plan this organization is on and the state of its billing relationship."
      needs={["GET /v1/billing/subscription"]}
      groups={[
        {
          title: "Subscription",
          items: [
            "current plan",
            "subscription status",
            "billing period",
            "renewal date",
            "seats",
            "payment status",
          ],
        },
        { title: "Plan limits", items: ["plan limits", "remaining quota"] },
      ]}
      note={SEPARATION_NOTE}
    />
  );
}

export function InvoicesBody() {
  return (
    <SpecView
      blurb="Issued invoices for the platform subscription."
      needs={["GET /v1/billing/invoices"]}
      groups={[
        {
          title: "Columns",
          items: [
            "invoice number",
            "period start",
            "period end",
            "amount",
            "status",
            "issued",
            "due",
            "paid",
          ],
        },
        { title: "Actions", items: ["download PDF", "download line items"] },
      ]}
      note="No invoice settlement exists yet, so this list can only ever be empty in practice. Provider AI spend is not invoiced here and does not appear on these documents."
    />
  );
}
