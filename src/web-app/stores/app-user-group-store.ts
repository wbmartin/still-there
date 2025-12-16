import { create } from "zustand";
import type { AppUserGroup, ApiAppUserGroupModel } from "../types/app-user-group-type";
import type { ApiResponse } from "../types/user-type";
import { apiRequest } from "../utils/api";
import { API_ENDPOINTS } from "../constants/api";

interface AppUserGroupState {
  appUserGroupList: AppUserGroup[];
  groupIdToName: Map<number, string>;
  maxAppUserGroupUpdateTs: number;
  loading: boolean;
  error: string | null;
  getGroupName: (id: number) => string;
  retrieveAppUserGroupList: () => Promise<void>;
  readGrantList: () => Promise<{ id: number; text: string }[]>;
  createUserGroup: (
    data: { groupName: string; userGrantList: number[] },
  ) => Promise<void>;
  updateUserGroup: (id: number, data: Partial<AppUserGroup>) => Promise<void>;
  deleteUserGroup: (id: number) => Promise<void>;
  clearCache: () => void;
}

// Helper to rebuild the group ID to name map
const rebuildGroupMap = (groups: AppUserGroup[]) => {
  const map = new Map<number, string>();
  groups.forEach(group => map.set(group.appUserGroupId, group.groupName));
  return map;
};

export const useAppUserGroupStore = create<AppUserGroupState>((set, get) => ({
      appUserGroupList: [],
  groupIdToName: new Map<number, string>(),
      maxAppUserGroupUpdateTs: 0,
  loading: false,
  error: null,
  getGroupName: (id: number) => get().groupIdToName.get(id) || `???${id}???`,
  //============================^FUNCTION LINE ^============================
  retrieveAppUserGroupList: async () => {
    set({ loading: true });
    try {
      const response = await apiRequest(API_ENDPOINTS.APP_USER_GROUP_LIST);
      if (!response.ok) throw new Error("Failed to fetch user groups");
      const data = await response.json() as ApiResponse<ApiAppUserGroupModel[]>;
      const appUserGroupList = data.data.map((g) => ({
        appUserGroupId: g.appUserGroupId,
        groupName: g.groupName,
        userGrantList: g.userGrantList,
        securityContext: g.securityContext,
        createTs: g.createTs,
        createUserId: g.createUserId,
        updateTs: g.updateTs,
        updateUserId: g.updateUserId,
      }));
      const groupIdToName = rebuildGroupMap(appUserGroupList);
      const maxAppUserGroupUpdateTs =
        appUserGroupList.length > 0
          ? Math.max(...appUserGroupList.map(g => g.updateTs))
          : 0;
      set({
        appUserGroupList,
        groupIdToName,
        maxAppUserGroupUpdateTs,
        loading: false
      });
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
      const response = await apiRequest(API_ENDPOINTS.APP_USER_GROUP, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(data),
      });
      if (!response.ok) throw new Error("Failed to create user group");
      // Refresh the list to get the new group with ID
      await get().retrieveAppUserGroupList();
    } catch (err: unknown) {
      set({ error: err instanceof Error ? err.message : "Unknown error" });
      throw err;
    }
  },
  //============================^FUNCTION LINE ^============================
   updateUserGroup: async (id: number, data: Partial<AppUserGroup>) => {
     set({ error: null });
     try {
       const state = get();
        const group = state.appUserGroupList.find((g) => g.appUserGroupId === id);
       if (!group) throw new Error("User group not found");
       const body = {
         appUserGroupId: id,
        groupName: data.groupName,
        userGrantList: data.userGrantList,
        updateTs: group.updateTs,
      };
      const response = await apiRequest(API_ENDPOINTS.APP_USER_GROUP, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
      if (!response.ok) {
        throw new Error("Failed to update user group");
      }
       // Update local state
        set((state) => {
          const updatedGroupList = state.appUserGroupList.map(
            (g: AppUserGroup) => (g.appUserGroupId === id ? { ...g, ...data } : g),
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
        const group = state.appUserGroupList.find((g) => g.appUserGroupId === id);
      if (!group) throw new Error("User group not found");
      const response = await apiRequest(API_ENDPOINTS.APP_USER_GROUP + `?appUserGroupId=${id}&updateTs=${group.updateTs}`, {
        method: "DELETE",
      });
      if (!response.ok) throw new Error("Failed to delete user group");
       // Remove from local state
        set((state) => {
          const updatedGroupList = state.appUserGroupList.filter(
            (g: AppUserGroup) => g.appUserGroupId !== id,
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
    const data = await response.json() as {
      data: { id: number; text: string }[]
    };
    return data.data;
  },
  //============================^FUNCTION LINE ^============================
  clearCache: () => {
    set({
  appUserGroupList: [],
      groupIdToName: new Map<number, string>(),
  maxAppUserGroupUpdateTs: 0,
    });
  },
}));