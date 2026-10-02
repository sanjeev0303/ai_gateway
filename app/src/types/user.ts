import type { ID, ISODateString } from "./common";
import type { OrganizationRole } from "./organization";

export type UserStatus = "active" | "invited" | "suspended" | "deactivated";

export interface User {
  id: ID;
  name: string;
  email: string;
  avatarUrl?: string;
  role: OrganizationRole;
  status: UserStatus;
  teamIds: ID[];
  projectIds: ID[];
  createdAt: ISODateString;
  lastActiveAt: ISODateString;
  /** Rolling 30-day spend attributed to this user, in whole USD. */
  spendMtd: number;
  /** True when the user bypassed SSO and holds a personal access token. */
  hasPersonalToken: boolean;
}
