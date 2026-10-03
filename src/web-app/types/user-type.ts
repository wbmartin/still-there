// User Types

/** app_user.account_status: a blocked login can neither sign in nor keep a
 *  session until an admin unblocks it (B034); an invited one has no
 *  password until its holder redeems the invite link (B035). */
export type AccountStatus = "active" | "blocked" | "invited";

export interface User {
  appUserId: number;
  userName: string;
  firstName: string;
  lastName: string;
  email: string;
  securityContext: number;
  appUserGroupMembershipList: number[];
  mfaExempt: boolean;
  accountStatus: AccountStatus;
  createTs: number;
  createUserId: number;
  updateTs: number;
  updateUserId: number;
}

/** POST /userInvite and /userInviteReissue: the token is shown once. */
export interface ApiInviteIssued {
  user?: User;
  inviteToken: string;
  expiresTs: number;
}

/** A freshly issued link, for the invite-link modal. */
export interface IssuedInvite {
  userName: string;
  inviteToken: string;
  expiresTs: number;
}

/** POST /inviteCheck: who a live link is for. */
export interface ApiInvitePreview {
  userName: string;
  firstName: string;
  expiresTs: number;
}

export interface UserFormData {
  userName: string;
  firstName: string;
  lastName: string;
  email: string;
  appUserGroupMembershipList: number[];
  mfaExempt: boolean;
  /** Create only: the raw password, hashed by the API, never stored. */
  password: string;
  updateTs: string;
}

export const defaultUserFormData: UserFormData = {
  userName: "",
  firstName: "",
  lastName: "",
  email: "",
  appUserGroupMembershipList: [],
  mfaExempt: false,
  password: "",
  updateTs: "",
};
