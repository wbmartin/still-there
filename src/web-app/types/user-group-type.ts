// User Group Types
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

export interface UserGroupFormData {
  groupName: string;
  userGrantList: number[];
  updateTs: string;
}

export const defaultUserGroupFormData: UserGroupFormData = {
  groupName: "",
  userGrantList: [],
  updateTs: "",
};