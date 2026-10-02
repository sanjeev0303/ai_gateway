import type { ID, ISODateString } from "./common";

export interface InvoiceLine {
  id: ID;
  description: string;
  quantity: number;
  /** Whole USD per unit. */
  unitPrice: number;
  amount: number;
}

export interface Invoice {
  id: ID;
  number: string;
  issuedAt: ISODateString;
  dueAt: ISODateString;
  status: "paid" | "open" | "past_due" | "void";
  subtotal: number;
  tax: number;
  total: number;
  lines: InvoiceLine[];
}

export interface PaymentMethod {
  id: ID;
  brand: "visa" | "mastercard" | "amex" | "ach";
  last4: string;
  expiresMonth: number;
  expiresYear: number;
  isDefault: boolean;
}

export interface UsageAllowance {
  /** Whole USD included in the current plan. */
  included: number;
  /** Whole USD consumed against that allowance. */
  used: number;
  overageRate: number;
  hardCap: number | null;
}

export interface BillingSummary {
  planName: string;
  /** Whole USD charged on the current recurring cycle. */
  monthlySubscription: number;
  /** Whole USD of metered usage this cycle. */
  meteredUsage: number;
  allowance: UsageAllowance;
  nextInvoiceDate: ISODateString;
  paymentMethod?: PaymentMethod;
  recentInvoices: Invoice[];
}
