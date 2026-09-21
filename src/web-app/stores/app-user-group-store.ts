import type { AppUserGroup } from "../types/app-user-group-type";
import type { ApiResponse } from "../types/api";
import { apiRequest, throwForFailedResponse } from "../utils/api";
import { API_ENDPOINTS } from "../constants/api";
import { createEntityStore } from "./create-entity-store";
import { readErrorMessage } from "../utils/api";

interface AppUserGroupExtras {
  // There is no GET /appUserGroup; a single group is read through the list
  // endpoint filtered to one id and merged into the cache.
  readAppUserGroupById: (id: number) => Promise<AppUserGroup | null>;
  getGroupName: (id: number) => string;
}

export const useAppUserGroupStore = createEntityStore<
  AppUserGroup,
  AppUserGroupExtras
>(
  {
    idParamName: "appUserGroupId",
    idField: "appUserGroupId",
    listEndpoint: API_ENDPOINTS.APP_USER_GROUP_LIST,
    itemEndpoint: API_ENDPOINTS.APP_USER_GROUP,
  },
  (set, get) => ({
    readAppUserGroupById: async (id: number) => {
      const searchParams = new URLSearchParams({
        appUserGroupIdList: String(id),
      });
      const response = await apiRequest(
        `${API_ENDPOINTS.APP_USER_GROUP_LIST}?${searchParams.toString()}`,
      );
      if (!response.ok) {
        await throwForFailedResponse(response, readErrorMessage);
      }
      const data = (await response.json()) as
        | ApiResponse<AppUserGroup[]>
        | null;
      const group = data?.data?.[0];
      if (!group) return null;
      // Refresh the cached row if it is present, otherwise add it. maxUpdateTs
      // is left alone; only a full list read advances it.
      const itemList = get().itemList;
      set({
        itemList: itemList.some((g) => g.appUserGroupId === group.appUserGroupId)
          ? itemList.map((g) =>
              g.appUserGroupId === group.appUserGroupId ? group : g,
            )
          : [...itemList, group],
      });
      return group;
    },
    getGroupName: (id: number) =>
      get().itemList.find((g) => g.appUserGroupId === id)?.groupName ??
      `???${id}???`,
  }),
);
