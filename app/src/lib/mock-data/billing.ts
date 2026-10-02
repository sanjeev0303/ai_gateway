import { NOW } from "./clock";
import { organization } from "./organization";
import { costSummary } from "./cost";
import type { BillingSummary, Invoice, InvoiceLine } from "@/types";

const daysAhead = (n: number) =>
  new Date(NOW.getTime() + n * 86_400_000).toISOString();
const daysAgo = (n: number) =>
  new Date(NOW.getTime() - n * 86_400_000).toISOString();

/** Platform fee mirrors the recorded MRR so the two never disagree. */
const SUBSCRIPTION = organization.mrr;

const line = (
  id: string,
  description: string,
  quantity: number,
  unitPrice: number,
): InvoiceLine => ({
  id,
  description,
  quantity,
  unitPrice,
  amount: Math.round(quantity * unitPrice * 100) / 100,
});

/** Most recent six billing periods; the current cycle is still open. */
const PERIODS: readonly { number: string; issuedDaysAgo: number; metered: number; status: Invoice["status"] }[] = [
  { number: "INV-2026-0042", issuedDaysAgo: 2, metered: costSummary.mtd, status: "open" },
  { number: "INV-2026-0038", issuedDaysAgo: 32, metered: costSummary.lastMonth, status: "paid" },
  { number: "INV-2026-0033", issuedDaysAgo: 63, metered: 14_820, status: "paid" },
  { number: "INV-2026-0029", issuedDaysAgo: 93, metered: 13_940, status: "paid" },
  { number: "INV-2026-0024", issuedDaysAgo: 124, metered: 12_610, status: "paid" },
  { number: "INV-2026-0020", issuedDaysAgo: 154, metered: 11_480, status: "paid" },
];

export const invoices: Invoice[] = PERIODS.map(
  ({ number, issuedDaysAgo, metered, status }, index) => {
    const issuedAt = daysAgo(issuedDaysAgo);
    const platform = line(
      `${number}_l1`,
      "Enterprise platform subscription",
      1,
      SUBSCRIPTION,
    );
    const usage = line(`${number}_l2`, "Metered model usage", 1, metered);
    const subtotal = Math.round((platform.amount + usage.amount) * 100) / 100;
    return {
      id: `inv_${index + 1}`,
      number,
      issuedAt,
      dueAt: daysAhead(28 - issuedDaysAgo),
      status,
      subtotal,
      // Acme Corp is VAT-registered in GB; reverse charge applies outside GB.
      tax: 0,
      total: subtotal,
      lines: [platform, usage],
    };
  },
);

export const billingSummary: BillingSummary = {
  planName: "Enterprise",
  monthlySubscription: SUBSCRIPTION,
  meteredUsage: costSummary.mtd,
  allowance: {
    included: 25_000,
    used: costSummary.mtd,
    overageRate: 1.1,
    hardCap: 30_000,
  },
  nextInvoiceDate: daysAhead(12),
  paymentMethod: {
    id: "pm_1",
    brand: "visa",
    last4: "4242",
    expiresMonth: 9,
    expiresYear: 2028,
    isDefault: true,
  },
  recentInvoices: invoices,
};
