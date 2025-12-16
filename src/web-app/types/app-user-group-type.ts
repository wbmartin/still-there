// App User Group Types
export interface AppUserGroup {
  appUserGroupId: number;
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

export interface ApiAppUserGroupModel {
  appUserGroupId: number;
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

export interface AppUserGroupFormData {
  groupName: string;
  userGrantList: number[];
  updateTs: string;
}

export const defaultAppUserGroupFormData: AppUserGroupFormData = {
  groupName: "",
  userGrantList: [],
  updateTs: "",
};