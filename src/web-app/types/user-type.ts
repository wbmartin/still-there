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

export interface UserGroup {
  userGroupId: number;
  groupName: string;
  userGrantList: number[];
  securityContext: number;
  createTs: number;
  createUserId: number;
  updateTs: number;
  updateUserId: number;
  deleteTs?: number;
  deleteUserId?: number;
}

export interface ApiUserGroupModel {
  userGroupId: number;
  groupName: string;
  userGrantList: number[];
  securityContext: number;
  createTs: number;
  createUserId: number;
  updateTs: number;
  updateUserId: number;
  deleteTs?: number;
  deleteUserId?: number;
}

export interface ApiResponse<T> {
  data: T;
}

export interface UserListResponse {
  users: User[];
}