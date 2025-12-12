import { create } from "zustand";
import type { User, ApiUserModel, ApiResponse } from "../types/user-type";
import { apiRequest } from "../utils/api";
import { API_ENDPOINTS } from "../constants/api";

interface AppUserState {
	users: User[];
	loading: boolean;
	error: string | null;
	readUserList: (idList?: number[]) => Promise<void>;
	createUser: (data: Partial<User>) => Promise<void>;
	updateUser: (id: number, data: Partial<User>) => Promise<void>;
	deleteUser: (id: number) => Promise<void>;
	addUserListToGroup: (userIds: number[], groupId: number) => Promise<void>;
	removeUserListFromGroup: (
		userIds: number[],
		groupId: number,
	) => Promise<void>;
	clearCache: () => void;
}

export const useAppUserStore = create<AppUserState>((set, get) => ({
  users: [],
  loading: false,
  error: null,
	//============================^FUNCTION LINE ^============================
	readUserList: async (idList?: number[]) => {
		set({ loading: true });
		try {
			let url = API_ENDPOINTS.USER_LIST;
			if (idList && idList.length > 0) {
				url += `?userIdList=${idList.join(',')}`;
			}
			const response = await apiRequest(url);
			if (!response.ok) throw new Error("Failed to fetch users");
			const data: ApiResponse<ApiUserModel[]> = await response.json();
			const users = data.data.map((u) => ({
				appUserId: u.appUserId,
				userName: u.userName,
				firstName: u.firstName,
				lastName: u.lastName,
				email: u.email,
				securityContext: u.securityContext,
				userGroupMembershipList: u.userGroupMembershipList,
				mfaExempt: u.mfaExempt,
				createTs: u.createTs,
				createUserId: u.createUserId,
				updateTs: u.updateTs,
				updateUserId: u.updateUserId,
			}));
			set({ users, loading: false });
		} catch (err: unknown) {
			const errorMessage = err instanceof Error ? err.message : "Unknown error";
			set({ loading: false, error: errorMessage });
		}
	},
  //============================^FUNCTION LINE ^============================
  createUser: async (data: Partial<User>) => {
    set({ error: null });
    try {
      const body = {
        userName: data.userName,
        firstName: data.firstName,
        lastName: data.lastName,
        email: data.email,
        userGroupMembershipList: data.userGroupMembershipList,
        mfaExempt: data.mfaExempt,
        pwHash: data.pwHash, // Assuming password is passed as pwHash
      };
      const response = await apiRequest(API_ENDPOINTS.USER, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
      if (!response.ok) throw new Error("Failed to create user");
      const result: ApiResponse<ApiUserModel> = await response.json();
      const apiUser = result.data;
      const newUser: User = {
        appUserId: apiUser.appUserId,
        userName: apiUser.userName,
        firstName: apiUser.firstName,
        lastName: apiUser.lastName,
        email: apiUser.email,
        securityContext: apiUser.securityContext,
        userGroupMembershipList: apiUser.userGroupMembershipList,
        mfaExempt: apiUser.mfaExempt,
        createTs: apiUser.createTs,
        createUserId: apiUser.createUserId,
        updateTs: apiUser.updateTs,
        updateUserId: apiUser.updateUserId,
      };
      set((state) => ({
        users: [...state.users, newUser],
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
        userGroupMembershipList: data.userGroupMembershipList,
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
        users: state.users.map(
          (u: any) => (u.appUserId === id ? { ...u, ...data } : u),
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
      const user = state.users.find((u) => u.appUserId === id);
      if (!user) throw new Error("User not found");
      const response = await apiRequest(API_ENDPOINTS.USER + `?userId=${id}&updateTs=${user.updateTs}`, {
        method: "DELETE",
      });
      if (!response.ok) throw new Error("Failed to delete user");
      // Remove from local state
      set((state) => ({
        users: state.users.filter((u: any) => u.appUserId !== id),
      }));
    } catch (err: unknown) {
      set({ error: err instanceof Error ? err.message : "Unknown error" });
      throw err;
    }
  },

  //============================^FUNCTION LINE ^============================
  addUserListToGroup: async (userIds: number[], groupId: number) => {
    set({ error: null });
    try {
      // Get current updateTs for each user
      const state = get();
      const updateTsList = userIds.map(id => {
        const user = state.users.find(u => u.appUserId === id);
        if (!user) throw new Error(`User ${id} not found in cache`);
        return user.updateTs;
      });

      const response = await apiRequest(API_ENDPOINTS.ADD_USER_LIST_TO_GROUP, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ userIds, groupId, updateTsList }),
      });
      if (!response.ok) throw new Error("Failed to add users to group");
      const data: ApiResponse<User[]> = await response.json();
      // Update local cache with returned users
      set((state) => ({
        users: state.users.map(u => {
          const updated = data.data.find(nu => nu.appUserId === u.appUserId);
          return updated || u;
        }),
      }));
    } catch (err: unknown) {
      const errorMessage = err instanceof Error ? err.message : "Unknown error";
      set({ error: errorMessage });
      throw new Error(errorMessage);
    }
  },

  //============================^FUNCTION LINE ^============================
  removeUserListFromGroup: async (userIds: number[], groupId: number) => {
    set({ error: null });
    try {
      // Get current updateTs for each user
      const state = get();
      const updateTsList = userIds.map(id => {
        const user = state.users.find(
          u => u.appUserId === id,
        );
        if (!user) throw new Error(`User ${id} not found in cache`);
        return user.updateTs;
      });

      const response = await apiRequest(
        API_ENDPOINTS.REMOVE_USER_LIST_FROM_GROUP,
        {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ userIds, groupId, updateTsList }),
      });
      if (!response.ok) throw new Error("Failed to remove users from group");
      const data: ApiResponse<User[]> = await response.json();
      // Update local cache with returned users
      set((state) => ({
        users: state.users.map(u => {
          const updated = data.data.find(nu => nu.appUserId === u.appUserId);
          return updated || u;
        }),
      }));
    } catch (err: unknown) {
      const errorMessage = err instanceof Error ? err.message : "Unknown error";
      set({ error: errorMessage });
      throw new Error(errorMessage);
    }
  },

  //============================^FUNCTION LINE ^============================
  clearCache: () => {
    set({ users: [] });
  },
}));