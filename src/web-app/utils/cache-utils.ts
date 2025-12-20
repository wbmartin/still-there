// Centralized cache clearing utility
// This ensures all application caches are cleared consistently

import { useExamplesStore } from "../stores/examples-store";
import { useAppUserStore } from "../stores/app-user-store";
import { useAppUserGroupStore } from "../stores/app-user-group-store";

export const clearAllCaches = () => {
  useExamplesStore.getState().clearCache();
  useAppUserStore.getState().clearCache();
  useAppUserGroupStore.getState().clearCache();
  // Add new stores here as they are added to the application
};