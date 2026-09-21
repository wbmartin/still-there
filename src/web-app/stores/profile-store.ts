import { create } from "zustand";
import type { ApiResponse } from "../types/api";
import type { Profile, ProfileUpdate } from "../types/profile-type";
import { apiJson, jsonBody } from "../utils/api";
import { API_ENDPOINTS } from "../constants/api";
import { registerCacheClearer } from "../utils/cache-utils";

// The signed-in user's own row. Loaded once the session is known (app.tsx)
// so preferences apply everywhere; cleared with every other cache on
// sign-out and security-context switches.
interface ProfileStore {
  profile: Profile | null;
  loading: boolean;
  error: string | null;
  readProfile: () => Promise<Profile>;
  updateProfile: (update: ProfileUpdate) => Promise<Profile>;
  clear: () => void;
}

export const useProfileStore = create<ProfileStore>((set) => ({
  profile: null,
  loading: false,
  error: null,

  readProfile: async () => {
    set({ loading: true, error: null });
    try {
      const response = await apiJson<ApiResponse<Profile>>(API_ENDPOINTS.ME);
      set({ profile: response.data, loading: false });
      return response.data;
    } catch (err) {
      set({
        loading: false,
        error: err instanceof Error ? err.message : "Failed to load profile",
      });
      throw err;
    }
  },

  updateProfile: async (update) => {
    set({ loading: true, error: null });
    try {
      const response = await apiJson<ApiResponse<Profile>>(API_ENDPOINTS.ME, {
        method: "PUT",
        ...jsonBody(update),
      });
      set({ profile: response.data, loading: false });
      return response.data;
    } catch (err) {
      set({
        loading: false,
        error: err instanceof Error ? err.message : "Failed to save profile",
      });
      throw err;
    }
  },

  clear: () => set({ profile: null, loading: false, error: null }),
}));

registerCacheClearer(() => useProfileStore.getState().clear());
