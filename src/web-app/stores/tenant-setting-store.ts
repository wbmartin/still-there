import { create } from "zustand";
import type { ApiResponse } from "../types/api";
import type { TenantSetting } from "../types/tenant-setting-type";
import { apiJson, isApiStatus, jsonBody } from "../utils/api";
import { API_ENDPOINTS } from "../constants/api";
import { registerCacheClearer } from "../utils/cache-utils";

export interface TenantSettingUpdate {
  timeZone: string;
  remindIntervalMin: number;
  dailyReminderTime: string;
  // 0 on the first save, which creates the row.
  updateTs: number;
}

// The tenant's one settings row: when and how often reminders go out.
interface TenantSettingStore {
  tenantSetting: TenantSetting | null;
  // GET /tenantSetting. null when it was never saved (the API's 404).
  readTenantSetting: () => Promise<TenantSetting | null>;
  // PUT /tenantSetting, optimistic lock on updateTs.
  saveTenantSetting: (update: TenantSettingUpdate) => Promise<TenantSetting>;
  clear: () => void;
}

export const useTenantSettingStore = create<TenantSettingStore>((set) => ({
  tenantSetting: null,

  readTenantSetting: async () => {
    let setting: TenantSetting;
    try {
      ({ data: setting } = await apiJson<ApiResponse<TenantSetting>>(
        API_ENDPOINTS.TENANT_SETTING,
      ));
    } catch (err) {
      if (isApiStatus(err, 404)) {
        set({ tenantSetting: null });
        return null;
      }
      throw err;
    }
    set({ tenantSetting: setting });
    return setting;
  },

  saveTenantSetting: async (update: TenantSettingUpdate) => {
    const { data: saved } = await apiJson<ApiResponse<TenantSetting>>(
      API_ENDPOINTS.TENANT_SETTING,
      { method: "PUT", ...jsonBody(update) },
    );
    set({ tenantSetting: saved });
    return saved;
  },

  clear: () => set({ tenantSetting: null }),
}));

registerCacheClearer(() => useTenantSettingStore.getState().clear());
