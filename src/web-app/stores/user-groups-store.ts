import { create } from "zustand";
import type { UserGroup, ApiUserGroupModel, ApiResponse } from "../types/user-type";
import { apiRequest } from "../utils/api";

interface UserGroupsState {
  userGroups: UserGroup[];
  loading: boolean;
  error: string | null;
  getUserGroupList: () => Promise<void>;
  clearCache: () => void;
}

export const useUserGroupsStore = create<UserGroupsState>((set) => ({
  userGroups: [],
  loading: false,
  error: null,
  //============================^FUNCTION LINE ^============================
  getUserGroupList: async () => {
    set({ loading: true });
    try {
      const response = await apiRequest("/userGroupList");
      if (!response.ok) throw new Error("Failed to fetch user groups");
      const data: ApiResponse<ApiUserGroupModel[]> = await response.json();
      const userGroups = data.data.map((g) => ({
        userGroupId: g.userGroupId,
        groupName: g.groupName,
        userGrantList: g.userGrantList,
        securityContext: g.securityContext,
        createTs: g.createTs,
        createUserId: g.createUserId,
        updateTs: g.updateTs,
        updateUserId: g.updateUserId,
      }));
      set({ userGroups, loading: false });
    } catch (err: unknown) {
      const errorMessage = err instanceof Error ? err.message : "Unknown error";
      set({ loading: false, error: errorMessage });
    }
  },
  //============================^FUNCTION LINE ^============================
  clearCache: () => {
    set({ userGroups: [] });
  },
}));