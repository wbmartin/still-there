// User Types

/** app_user.account_status: a blocked login can neither sign in nor keep a
 *  session until an admin unblocks it (B034). */
export type AccountStatus = "active" | "blocked";

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
  pwHash?: string;
  createTs: number;
  createUserId: number;
  updateTs: number;
  updateUserId: number;
}




export interface UserFormData {
  userName: string;
  firstName: string;
  lastName: string;
  email: string;
  appUserGroupMembershipList: number[];
  mfaExempt: boolean;
  pwHash: string;
  updateTs: string;
}

export const defaultUserFormData: UserFormData = {
  userName: "",
  firstName: "",
  lastName: "",
  email: "",
  appUserGroupMembershipList: [],
  mfaExempt: false,
  pwHash: "",
  updateTs: "",
};
