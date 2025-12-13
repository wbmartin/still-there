// User Types
export interface User {
  appUserId: number;
  userName: string;
  firstName: string;
  lastName: string;
  email: string;
  securityContext: number;
  userGroupMembershipList: number[];
  mfaExempt: boolean;
  pwHash?: string;
  createTs: number;
  createUserId: number;
  updateTs: number;
  updateUserId: number;
}

export interface ApiUserModel {
  appUserId: number;
  userName: string;
  firstName: string;
  lastName: string;
  email: string;
  securityContext: number;
  userGroupMembershipList: number[];
  mfaExempt: boolean;
  createTs: number;
  createUserId: number;
  updateTs: number;
  updateUserId: number;
}

export interface ApiResponse<T> {
  data: T;
}

export interface UserListResponse {
  users: User[];
}

export interface UserFormData {
  userName: string;
  firstName: string;
  lastName: string;
  email: string;
  userGroupMembershipList: number[];
  mfaExempt: boolean;
  pwHash: string;
  updateTs: string;
}

export const defaultUserFormData: UserFormData = {
  userName: "",
  firstName: "",
  lastName: "",
  email: "",
  userGroupMembershipList: [],
  mfaExempt: false,
  pwHash: "",
  updateTs: "",
};