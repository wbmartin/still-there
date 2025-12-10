import { create } from "zustand";
import type { User, ApiUserModel, ApiResponse } from "../types/user-type";
import { apiRequest } from "../utils/api";
import { API_ENDPOINTS } from "../constants/api";

interface UserState {
  users: User[];
  loading: boolean;
  error: string | null;
  readUserList: () => Promise<void>;
  createUser: (data: Partial<User>) => Promise<void>;
  updateUser: (id: number, data: Partial<User>) => Promise<void>;
  deleteUser: (id: number) => Promise<void>;
  clearCache: () => void;
}

export const useUserStore = create<UserState>((set, get) => ({
  users: [],
  loading: false,
  error: null,
  //============================^FUNCTION LINE ^============================
  readUserList: async () => {
    set({ loading: true });
    try {
      const response = await apiRequest(API_ENDPOINTS.USER_LIST);
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
      if (!response.ok) throw new Error("Failed to update user");
      // Update local state
      set((state) => ({
        users: state.users.map((u: any) => (u.appUserId === id ? { ...u, ...data } : u)),
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
  clearCache: () => {
    set({ users: [] });
  },
}));