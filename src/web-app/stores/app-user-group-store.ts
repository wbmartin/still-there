import type { AppUserGroup } from "../types/app-user-group-type";
import { API_ENDPOINTS } from "../constants/api";
import { createEntityStore } from "./create-entity-store";

interface AppUserGroupExtras {
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
  (_set, get) => ({
    getGroupName: (id: number) =>
      get().itemList.find((g) => g.appUserGroupId === id)?.groupName ??
      `???${id}???`,
  }),
);
