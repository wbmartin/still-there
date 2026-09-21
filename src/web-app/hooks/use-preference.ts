import { useProfileStore } from "../stores/profile-store";
import { resolvePreference } from "../types/profile-type";

// The signed-in user's preferences with defaults applied. Before the
// profile has loaded (or when the user never saved any) these are the
// defaults, so a page can read them unconditionally.
export const usePreference = () =>
  resolvePreference(
    useProfileStore((state) => state.profile?.preferenceJsonb),
  );

export const useRowsPerPage = () => usePreference().rowsPerPage;
