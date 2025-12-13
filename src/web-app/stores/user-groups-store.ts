import { create } from "zustand";
import type { UserGroup, ApiUserGroupModel } from "../types/user-group-type";
import type { ApiResponse } from "../types/user-type";
import { apiRequest } from "../utils/api";
import { API_ENDPOINTS } from "../constants/api";

interface UserGroupsState {
  userGroupList: UserGroup[];
  groupIdToName: Map<number, string>;
  maxUserGroupUpdateTs: number;
  loading: boolean;
  error: string | null;
  getGroupName: (id: number) => string;
  readUserGroupList: () => Promise<void>;
  readGrantList: () => Promise<{ id: number; text: string }[]>;
  createUserGroup: (
    data: { groupName: string; userGrantList: number[] },
  ) => Promise<void>;
  updateUserGroup: (id: number, data: Partial<UserGroup>) => Promise<void>;
  deleteUserGroup: (id: number) => Promise<void>;
  clearCache: () => void;
}

// Helper to rebuild the group ID to name map
const rebuildGroupMap = (groups: UserGroup[]) => {
  const map = new Map<number, string>();
  groups.forEach(group => map.set(group.userGroupId, group.groupName));
  return map;
};

export const useUserGroupsStore = create<UserGroupsState>((set, get) => ({
  userGroupList: [],
  groupIdToName: new Map<number, string>(),
  maxUserGroupUpdateTs: 0,
  loading: false,
  error: null,
  getGroupName: (id: number) => get().groupIdToName.get(id) || `???${id}???`,
  //============================^FUNCTION LINE ^============================
  readUserGroupList: async () => {
    set({ loading: true });
    try {
      const response = await apiRequest(API_ENDPOINTS.USER_GROUP_LIST);
      if (!response.ok) throw new Error("Failed to fetch user groups");
      const data = await response.json() as ApiResponse<ApiUserGroupModel[]>;
      const userGroupList = data.data.map((g) => ({
        userGroupId: g.userGroupId,
        groupName: g.groupName,
        userGrantList: g.userGrantList,
        securityContext: g.securityContext,
        createTs: g.createTs,
        createUserId: g.createUserId,
        updateTs: g.updateTs,
        updateUserId: g.updateUserId,
      }));
      const groupIdToName = rebuildGroupMap(userGroupList);
      const maxUserGroupUpdateTs =
        userGroupList.length > 0 ? Math.max(...userGroupList.map(g => g.updateTs)) : 0;
      set({ userGroupList, groupIdToName, maxUserGroupUpdateTs, loading: false });
    } catch (err: unknown) {
      const errorMessage = err instanceof Error ? err.message : "Unknown error";
      set({ loading: false, error: errorMessage });
    }
   },
   //============================^FUNCTION LINE ^================
   createUserGroup: async (
     data: { groupName: string; userGrantList: number[] },
   ) => {
    set({ error: null });
    try {
      const response = await apiRequest(API_ENDPOINTS.USER_GROUP, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(data),
      });
      if (!response.ok) throw new Error("Failed to create user group");
      // Refresh the list to get the new group with ID
      await get().readUserGroupList();
    } catch (err: unknown) {
      set({ error: err instanceof Error ? err.message : "Unknown error" });
      throw err;
    }
  },
  //============================^FUNCTION LINE ^============================
   updateUserGroup: async (id: number, data: Partial<UserGroup>) => {
     set({ error: null });
     try {
       const state = get();
       const group = state.userGroupList.find((g) => g.userGroupId === id);
      if (!group) throw new Error("User group not found");
      const body = {
        userGroupId: id,
        groupName: data.groupName,
        userGrantList: data.userGrantList,
        updateTs: group.updateTs,
      };
      const response = await apiRequest(API_ENDPOINTS.USER_GROUP, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
      if (!response.ok) {
        throw new Error("Failed to update user group");
      }
       // Update local state
        set((state) => {
          const updatedGroupList = state.userGroupList.map(
            (g: UserGroup) => (g.userGroupId === id ? { ...g, ...data } : g),
          );
          const groupIdToName = rebuildGroupMap(updatedGroupList);
          return { userGroupList: updatedGroupList, groupIdToName };
        });
    } catch (err: unknown) {
      set({ error: err instanceof Error ? err.message : "Unknown error" });
    }
  },
  //============================^FUNCTION LINE ^============================
   deleteUserGroup: async (id: number) => {
     set({ error: null });
     try {
       const state = get();
       const group = state.userGroupList.find((g) => g.userGroupId === id);
      if (!group) throw new Error("User group not found");
      const response = await apiRequest(API_ENDPOINTS.USER_GROUP + `?userGroupId=${id}&updateTs=${group.updateTs}`, {
        method: "DELETE",
      });
      if (!response.ok) throw new Error("Failed to delete user group");
       // Remove from local state
        set((state) => {
          const updatedGroupList = state.userGroupList.filter(
            (g: UserGroup) => g.userGroupId !== id,
          );
          const groupIdToName = rebuildGroupMap(updatedGroupList);
          return { userGroupList: updatedGroupList, groupIdToName };
        });
    } catch (err: unknown) {
      set({ error: err instanceof Error ? err.message : "Unknown error" });
      throw err;
    }
  },
  //============================^FUNCTION LINE ^================
  readGrantList: async () => {
      const response = await apiRequest(
        API_ENDPOINTS.GRANT_LIST,
      );
    if (!response.ok) throw new Error("Failed to fetch grant list");
    const data = await response.json() as { data: { id: number; text: string }[] };
    return data.data;
  },
  //============================^FUNCTION LINE ^============================
  clearCache: () => {
    set({
      userGroupList: [],
      groupIdToName: new Map<number, string>(),
      maxUserGroupUpdateTs: 0,
    });
  },
}));