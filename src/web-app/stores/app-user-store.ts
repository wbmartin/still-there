import type { User } from "../types/user-type";
import type { ApiResponse } from "../types/api";
import { apiJson, apiRequest, jsonBody, throwForFailedResponse } from "../utils/api";
import { API_ENDPOINTS, type ApiEndpoint } from "../constants/api";
import { createEntityStore } from "./create-entity-store";
import { readErrorMessage } from "../utils/api";

interface AppUserExtras {
  // There is no GET /user; a single user is read through the list endpoint
  // with afterUpdateTs so an unchanged row costs nothing and the cached copy
  // is returned instead.
  readUserById: (id: number) => Promise<User | null>;
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
  changeOtherUserPassword: (
    appUserId: number,
    newPassword: string,
  ) => Promise<void>;
  resetUserMFA: (userId: number, updateTs: number) => Promise<void>;
  // Block (true) or unblock (false) a login by name; the key is the
  // handler's json tag (model.BlockUserRequest).
  setUserBlocked: (userName: string, blocked: boolean) => Promise<void>;
}

export const useAppUserStore = createEntityStore<User, AppUserExtras>(
  {
    idParamName: "appUserId",
    idField: "appUserId",
    listEndpoint: API_ENDPOINTS.USER_LIST,
    itemEndpoint: API_ENDPOINTS.USER,
  },
  (set, get) => {
    const mergeUserList = (userList: User[]) => {
      set({
        itemList: userList.reduce(
          (list, updated) =>
            list.some((u) => u.appUserId === updated.appUserId)
              ? list.map((u) =>
                  u.appUserId === updated.appUserId ? updated : u,
                )
              : [...list, updated],
          get().itemList,
        ),
      });
    };

    const postUserListGroupChange = async (
      endpoint: string,
      body: { userIds: number[]; groupId: number; updateTsList: number[] },
    ) => {
      set({ error: null });
      const response = await apiRequest(endpoint, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
      if (!response.ok) {
        await throwForFailedResponse(response, readErrorMessage);
      }
      const data = (await response.json()) as ApiResponse<User[]>;
      mergeUserList(data.data);
    };

    const postOrThrow = async (endpoint: ApiEndpoint, body: unknown) => {
      set({ error: null });
      try {
        await apiJson<unknown>(endpoint, { method: "POST", ...jsonBody(body) });
      } catch (err) {
        set({ error: err instanceof Error ? err.message : "Request failed" });
        throw err;
      }
    };

    return {
      readUserById: async (id: number) => {
        const searchParams = new URLSearchParams({
          appUserIdList: String(id),
          afterUpdateTs: String(get().maxUpdateTs),
        });
        const response = await apiRequest(
          `${API_ENDPOINTS.USER_LIST}?${searchParams.toString()}`,
        );
        if (!response.ok) {
          await throwForFailedResponse(response, readErrorMessage);
        }
        const data = (await response.json()) as ApiResponse<User[]> | null;
        if (data?.data && data.data.length > 0) {
          mergeUserList(data.data);
        }
        return get().itemList.find((u) => u.appUserId === id) ?? null;
      },

      addUserListToGroup: (userIds, groupId, updateTsList) =>
        postUserListGroupChange(API_ENDPOINTS.ADD_USER_LIST_TO_GROUP, {
          userIds,
          groupId,
          updateTsList,
        }),

      removeUserListFromGroup: (userIds, groupId, updateTsList) =>
        postUserListGroupChange(API_ENDPOINTS.REMOVE_USER_LIST_FROM_GROUP, {
          userIds,
          groupId,
          updateTsList,
        }),

      // The key is the handler's json tag; tests/app-user-store.test.ts.
      changeOtherUserPassword: (appUserId, newPassword) =>
        postOrThrow(API_ENDPOINTS.CHANGE_OTHER_USER_PASSWORD, {
          appUserId,
          newPassword,
        }),

      resetUserMFA: (userId, updateTs) =>
        postOrThrow(API_ENDPOINTS.ADMIN_RESET_USER_MFA, { userId, updateTs }),

      setUserBlocked: (userName, blocked) =>
        postOrThrow(
          blocked
            ? API_ENDPOINTS.ADMIN_BLOCK_USER
            : API_ENDPOINTS.ADMIN_UNBLOCK_USER,
          { userName },
        ),
    };
  },
);
