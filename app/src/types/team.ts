import type { ID, ISODateString } from "./common";

export interface Team {
  id: ID;
  name: string;
  slug: string;
  description: string;
  memberIds: ID[];
  leadId: ID;
  projectIds: ID[];
  createdAt: ISODateString;
  /** Spend budget in whole USD, or null when the team is unbudgeted. */
  monthlyBudget: number | null;
  spendMtd: number;
}
