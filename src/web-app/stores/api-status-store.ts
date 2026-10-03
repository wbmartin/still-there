// "Is the API answering?" -- one flag for the whole app.
//
// A screen fires several calls, so a stopped API produces several failures
// at once. This store turns all of them into one banner (app.tsx, shown
// above the routed page while signed in), and the first call that succeeds
// clears it.
//
// It is one of the two subscribers to subscribeApiOutcome; the other is the
// auth store. The subscription is installed here, at module load, so
// importing the store is all a screen has to do.
import { create } from "zustand";
import { subscribeApiOutcome, type ApiUnreachableError } from "../utils/api";

interface ApiStatusState {
  /** True from the first ApiUnreachableError until the next 2xx. */
  unreachable: boolean;
  /** The message of the failure that raised the flag. */
  reason: string | null;
  markUnreachable: (error: ApiUnreachableError) => void;
  markReachable: () => void;
}

export const useApiStatusStore = create<ApiStatusState>((set, get) => ({
  unreachable: false,
  reason: null,
  // Both guards are the "one banner, not one per call" rule: state changes
  // on the transition only, so ten failing calls render the banner once and
  // every later success is free.
  markUnreachable: (error) => {
    if (get().unreachable) return;
    set({ unreachable: true, reason: error.message });
  },
  markReachable: () => {
    if (!get().unreachable) return;
    set({ unreachable: false, reason: null });
  },
}));

subscribeApiOutcome((outcome) => {
  if (outcome.kind === "unreachable") {
    useApiStatusStore.getState().markUnreachable(outcome.error);
    return;
  }
  if (outcome.kind === "ok") {
    useApiStatusStore.getState().markReachable();
  }
  // "session-expired" is the auth store's business, not the banner's.
});
