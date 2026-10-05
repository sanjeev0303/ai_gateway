import type { ID, ISODateString } from "./common";
import type { OrganizationRole } from "./organization";

/** Mirrors `domain.UserStatus*`. Invited people are rows in `/v1/invitations`,
 *  not members, so "invited" is not a member status. */
export type UserStatus = "active" | "disabled";

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
}
