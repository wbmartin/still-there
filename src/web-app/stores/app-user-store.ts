import type {
  ApiInviteIssued,
  IssuedInvite,
  User,
} from "../types/user-type";
import type { ApiResponse } from "../types/api";
import { apiJson, jsonBody } from "../utils/api";
import { API_ENDPOINTS, type ApiEndpoint } from "../constants/api";
import { createEntityStore } from "./create-entity-store";

interface AppUserExtras {
  // There is no GET /user; a single user is read through the list endpoint
  // with afterUpdateTs so an unchanged row costs nothing and the cached copy
  // is returned instead.
  readUserById: (id: number) => Promise<User | null>;
  addUserListToGroup: (
    userIdList: number[],
    appUserGroupId: number,
    updateTsList: number[],
  ) => Promise<void>;
  removeUserListFromGroup: (
    userIdList: number[],
    appUserGroupId: number,
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
  // Invite links (B035): create an invited login with no password and get
  // its one-time link; replace a lost link; stop a link working. The token
  // comes back once, from the issuing call, and is never stored here.
  inviteUser: (data: InviteUserData) => Promise<IssuedInvite>;
  reissueInvite: (
    appUserId: number,
    userName: string,
  ) => Promise<IssuedInvite>;
  revokeInvite: (appUserId: number) => Promise<void>;
}

/** POST /userInvite: the POST /user fields without a password. */
export interface InviteUserData {
  userName: string;
  firstName: string;
  lastName: string;
  email: string;
  appUserGroupMembershipList: number[];
  mfaExempt: boolean;
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
      // Keys are the handler's json tags (userIdList, appUserGroupId).
      body: {
        userIdList: number[];
        appUserGroupId: number;
        updateTsList: number[];
      },
    ) => {
      set({ error: null });
      const data = await apiJson<ApiResponse<User[]>>(endpoint, {
        method: "POST",
        ...jsonBody(body),
      });
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
        const data = await apiJson<ApiResponse<User[]> | null | undefined>(
          `${API_ENDPOINTS.USER_LIST}?${searchParams.toString()}`,
        );
        if (data?.data && data.data.length > 0) {
          mergeUserList(data.data);
        }
        return get().itemList.find((u) => u.appUserId === id) ?? null;
      },

      addUserListToGroup: (userIdList, appUserGroupId, updateTsList) =>
        postUserListGroupChange(API_ENDPOINTS.ADD_USER_LIST_TO_GROUP, {
          userIdList,
          appUserGroupId,
          updateTsList,
        }),

      removeUserListFromGroup: (userIdList, appUserGroupId, updateTsList) =>
        postUserListGroupChange(API_ENDPOINTS.REMOVE_USER_LIST_FROM_GROUP, {
          userIdList,
          appUserGroupId,
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

      inviteUser: async (data) => {
        set({ error: null });
        try {
          const result = await apiJson<ApiResponse<ApiInviteIssued>>(
            API_ENDPOINTS.USER_INVITE,
            { method: "POST", ...jsonBody(data) },
          );
          if (result.data.user) mergeUserList([result.data.user]);
          return {
            userName: data.userName,
            inviteToken: result.data.inviteToken,
            expiresTs: result.data.expiresTs,
          };
        } catch (err) {
          set({ error: err instanceof Error ? err.message : "Request failed" });
          throw err;
        }
      },

      reissueInvite: async (appUserId, userName) => {
        const result = await apiJson<ApiResponse<ApiInviteIssued>>(
          API_ENDPOINTS.USER_INVITE_REISSUE,
          { method: "POST", ...jsonBody({ appUserId }) },
        );
        return {
          userName,
          inviteToken: result.data.inviteToken,
          expiresTs: result.data.expiresTs,
        };
      },

      revokeInvite: (appUserId) =>
        postOrThrow(API_ENDPOINTS.USER_INVITE_REVOKE, { appUserId }),
    };
  },
);
