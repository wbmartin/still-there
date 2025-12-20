import { create } from "zustand";
import type { User, ApiUserModel, ApiResponse } from "../types/user-type";
import { apiRequest } from "../utils/api";
import { API_ENDPOINTS } from "../constants/api";

interface AppUserState {
  userList: User[];
  maxAppUserUpdateTs: number;
  loading: boolean;
  error: string | null;
  retrieveUserList: () => Promise<void>;
  retrieveUserById: (id: number) => Promise<User | null>;
  addUserListToGroup: (
    userIds: number[],
    groupId: number,
    updateTsList: number[],
  ) => Promise<void>;
  removeUserListFromGroup: (
    userIds: number[],
    groupId: number,
    updateTsList: number[],
  ) => Promise<void>;
  createUser: (
    data: Omit<
      User,
      | "appUserId"
      | "createTs"
      | "createUserId"
      | "updateTs"
      | "updateUserId"
      | "securityContext"
    >,
  ) => Promise<void>;
  updateUser: (id: number, data: Partial<User>) => Promise<void>;
  deleteUser: (id: number) => Promise<void>;
  changeOtherUserPassword: (
    userId: number,
    newPassword: string,
  ) => Promise<void>;
  resetUserMFA: (userId: number, updateTs: number) => Promise<void>;
  clearCache: () => void;
}

export const useAppUserStore = create<AppUserState>((set, get) => ({
  userList: [],
  maxAppUserUpdateTs: 0,
  loading: false,
  error: null,
  //============================^FUNCTION LINE ^============================
  retrieveUserList: async (idList?: number[]) => {
    set({ loading: true });
    try {
      let url = API_ENDPOINTS.USER_LIST;
      if (idList && idList.length > 0) {
        url += `?userIdList=${idList.join(",")}`;
      }
      const response = await apiRequest(url);
      if (!response.ok) throw new Error("Failed to fetch users");
      const data = (await response.json()) as ApiResponse<ApiUserModel[]>;
      const users = data.data.map((u) => ({
        appUserId: u.appUserId,
        userName: u.userName,
        firstName: u.firstName,
        lastName: u.lastName,
        email: u.email,
        securityContext: u.securityContext,
        appUserGroupMembershipList: u.appUserGroupMembershipList,
        mfaExempt: u.mfaExempt,
        createTs: u.createTs,
        createUserId: u.createUserId,
        updateTs: u.updateTs,
        updateUserId: u.updateUserId,
      }));
      set({ userList: users, loading: false });

      // Update maxAppUserUpdateTs only for non-ID-specific fetches
      if (!idList || idList.length === 0) {
        const newMax =
          users.length > 0 ? Math.max(...users.map((u) => u.updateTs)) : 0;
        set({ userList: users, maxAppUserUpdateTs: newMax });
      }
    } catch (err: unknown) {
      const errorMessage = err instanceof Error ? err.message : "Unknown error";
      set({ loading: false, error: errorMessage });
    }
  },
  retrieveUserById: async (id: number) => {
    const maxTs = get().maxAppUserUpdateTs;
    const url = `${API_ENDPOINTS.USER_LIST}?userIdList=${id}&afterUpdateTs=${maxTs}`;
    try {
      const response = await apiRequest(url);
      if (!response.ok) throw new Error("Failed to fetch user");
      const data = (await response.json()) as ApiResponse<
        ApiUserModel[]
      > | null;
      if (!data || !data.data) {
        return get().userList.find((u) => u.appUserId === id) || null;
      }
      if (data.data.length > 0) {
        // Update cache with new user data (sync)
        const updatedUser = {
          appUserId: data.data[0].appUserId,
          userName: data.data[0].userName,
          firstName: data.data[0].firstName,
          lastName: data.data[0].lastName,
          email: data.data[0].email,
          securityContext: data.data[0].securityContext,
          appUserGroupMembershipList: data.data[0].appUserGroupMembershipList,
          mfaExempt: data.data[0].mfaExempt,
          createTs: data.data[0].createTs,
          createUserId: data.data[0].createUserId,
          updateTs: data.data[0].updateTs,
          updateUserId: data.data[0].updateUserId,
        };
        set((state) => ({
          userList: state.userList.some((u) => u.appUserId === id)
            ? state.userList.map((u) => (u.appUserId === id ? updatedUser : u))
            : [...state.userList, updatedUser],
        }));
        return updatedUser;
      } else {
        // Return cached user
        return get().userList.find((u) => u.appUserId === id) || null;
      }
    } catch (err: unknown) {
      const errorMessage = err instanceof Error ? err.message : "Unknown error";
      set({ error: errorMessage });
      return null;
    }
  },
  addUserListToGroup: async (
    userIds: number[],
    groupId: number,
    updateTsList: number[],
  ) => {
    const response = await apiRequest(API_ENDPOINTS.ADD_USER_LIST_TO_GROUP, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ userIds, groupId, updateTsList }),
    });
    if (!response.ok) throw new Error("Failed to add users to group");
    const data = (await response.json()) as ApiResponse<ApiUserModel[]>;
    // Update the cache with the returned users
    set((state) => ({
      userList: state.userList.map((u) => {
        const updated = data.data.find((du) => du.appUserId === u.appUserId);
        return updated
          ? {
              ...u,
              appUserGroupMembershipList: updated.appUserGroupMembershipList,
              updateTs: updated.updateTs,
            }
          : u;
      }),
    }));
  },

  createUser: async (data: Partial<User>) => {
    set({ error: null });
    try {
      const body = {
        userName: data.userName,
        firstName: data.firstName,
        lastName: data.lastName,
        email: data.email,
        appUserGroupMembershipList: data.appUserGroupMembershipList,
        mfaExempt: data.mfaExempt,
        pwHash: data.pwHash,
      };
      const response = await apiRequest(API_ENDPOINTS.USER, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
      if (!response.ok) throw new Error("Failed to create user");
      const result = (await response.json()) as ApiResponse<ApiUserModel>;
      const apiUser = result.data;
      const newUser: User = {
        appUserId: apiUser.appUserId,
        userName: apiUser.userName,
        firstName: apiUser.firstName,
        lastName: apiUser.lastName,
        email: apiUser.email,
        securityContext: apiUser.securityContext,
        appUserGroupMembershipList: apiUser.appUserGroupMembershipList,
        mfaExempt: apiUser.mfaExempt,
        createTs: apiUser.createTs,
        createUserId: apiUser.createUserId,
        updateTs: apiUser.updateTs,
        updateUserId: apiUser.updateUserId,
      };
      set((state) => ({
        userList: [...state.userList, newUser],
      }));
    } catch (err: unknown) {
      const errorMessage = err instanceof Error ? err.message : "Unknown error";
      set({ error: errorMessage });
      throw new Error(errorMessage);
    }
  },
  //============================^FUNCTION LINE ^============================
  updateUser: async (id: number, data: Partial<User>) => {
    set({ error: null });
    try {
      const body = {
        appUserId: id,
        userName: data.userName,
        firstName: data.firstName,
        lastName: data.lastName,
        email: data.email,
        appUserGroupMembershipList: data.appUserGroupMembershipList,
        mfaExempt: data.mfaExempt,
        updateTs: data.updateTs,
      };
      const response = await apiRequest(API_ENDPOINTS.USER, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
      if (!response.ok) {
        throw new Error("Failed to update user");
      }
      // Update local state
      set((state) => ({
        userList: state.userList.map((u: User) =>
          u.appUserId === id ? { ...u, ...data } : u,
        ),
      }));
    } catch (err: unknown) {
      set({ error: err instanceof Error ? err.message : "Unknown error" });
    }
  },
  //============================^FUNCTION LINE ^============================
  deleteUser: async (id: number) => {
    set({ error: null });
    try {
      const state = get();
      const user = state.userList.find((u) => u.appUserId === id);
      if (!user) throw new Error("User not found");
      const response = await apiRequest(
        API_ENDPOINTS.USER + `?userId=${id}&updateTs=${user.updateTs}`,
        {
          method: "DELETE",
        },
      );
      if (!response.ok) throw new Error("Failed to delete user");
      // Remove from local state
      set((state) => ({
        userList: state.userList.filter((u: User) => u.appUserId !== id),
      }));
    } catch (err: unknown) {
      set({ error: err instanceof Error ? err.message : "Unknown error" });
      throw err;
    }
  },

  //============================^FUNCTION LINE ^============================
  removeUserListFromGroup: async (
    userIds: number[],
    groupId: number,
    updateTsList: number[],
  ) => {
    set({ error: null });
    try {
      const response = await apiRequest(
        API_ENDPOINTS.REMOVE_USER_LIST_FROM_GROUP,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ userIds, groupId, updateTsList }),
        },
      );
      if (!response.ok) throw new Error("Failed to remove users from group");
      const data = (await response.json()) as ApiResponse<User[]>;
      // Update local cache with returned users
      set((state) => ({
        userList: state.userList.map(
          (u) => data.data.find((nu) => nu.appUserId === u.appUserId) || u,
        ),
      }));
    } catch (err: unknown) {
      const errorMessage = err instanceof Error ? err.message : "Unknown error";
      set({ error: errorMessage });
      throw new Error(errorMessage);
    }
  },

  changeOtherUserPassword: async (userId: number, newPassword: string) => {
    set({ error: null });
    try {
      console.log(
        "Changing password for userId:",
        userId,
        "to newPassword:",
        newPassword,
      );
      const response = await apiRequest(
        API_ENDPOINTS.CHANGE_OTHER_USER_PASSWORD,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ userId, newPassword }),
        },
      );
      if (!response.ok) {
        const errorData = (await response.json()) as { error?: string };
        throw new Error(errorData.error || "Failed to change password");
      }
    } catch (err: unknown) {
      const errorMessage =
        err instanceof Error ? err.message : "Failed to change password";
      set({ error: errorMessage });
      throw new Error(errorMessage);
    }
  },

  resetUserMFA: async (userId: number, updateTs: number) => {
    set({ error: null });
    try {
      const response = await apiRequest("/admin/resetUserMFA", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ userId, updateTs }),
      });
      if (!response.ok) {
        const errorData = (await response.json()) as { error?: string };
        throw new Error(errorData.error || "Failed to reset MFA");
      }
    } catch (err: unknown) {
      const errorMessage =
        err instanceof Error ? err.message : "Failed to reset MFA";
      set({ error: errorMessage });
      throw new Error(errorMessage);
    }
  },

  clearCache: () => {
    set({ userList: [], maxAppUserUpdateTs: 0 });
  },
}));
