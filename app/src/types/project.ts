import type { ID, ISODateString } from "./common";

export type ProjectEnvironment = "production" | "staging" | "development";

export interface Project {
  id: ID;
  name: string;
  slug: string;
  description: string;
  environment: ProjectEnvironment;
  teamId: ID;
  ownerId: ID;
  status: "active" | "archived";
  createdAt: ISODateString;
  /** Ids of models this project is permitted to call. */
  allowedModelIds: ID[];
  spendMtd: number;
  monthlyBudget: number | null;
}
