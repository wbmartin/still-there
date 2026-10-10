import { create } from "zustand";
import { registerCacheClearer } from "../utils/cache-utils";

// The help module's UI state (doc/151-help-module.md): whether the Help
// panel is open and on which entry, and the running page tour. What a user
// has seen and finished is kept on their profile instead
// (help/help-preference.ts).
interface HelpStore {
  opened: boolean;
  // A glossary term or topic id; null is the panel's home for the page.
  entryId: string | null;
  // Entries opened before this one, for Back.
  historyList: string[];
  openHelp: (entryId?: string | null) => void;
  back: () => void;
  goHome: () => void;
  closeHelp: () => void;
  // The running tour and its step; -1 while it finds its first step.
  activeTourId: string | null;
  tourStep: number;
  startTour: (tourId: string) => void;
  setTourStep: (step: number) => void;
  endTour: () => void;
  clear: () => void;
}

const initialState = {
  opened: false,
  entryId: null,
  historyList: [],
  activeTourId: null,
  tourStep: 0,
};

export const useHelpStore = create<HelpStore>((set, get) => ({
  ...initialState,

  openHelp: (entryId = null) => {
    const { opened, entryId: current, historyList } = get();
    set({
      opened: true,
      entryId,
      historyList:
        opened && current && current !== entryId
          ? [...historyList, current]
          : historyList,
    });
  },
  back: () => {
    const historyList = [...get().historyList];
    const previous = historyList.pop() ?? null;
    set({ entryId: previous, historyList });
  },
  goHome: () => set({ entryId: null, historyList: [] }),
  closeHelp: () => set({ opened: false, entryId: null, historyList: [] }),

  startTour: (tourId) =>
    set({ activeTourId: tourId, tourStep: -1, opened: false }),
  setTourStep: (tourStep) => set({ tourStep }),
  endTour: () => set({ activeTourId: null, tourStep: 0 }),

  clear: () => set(initialState),
}));

registerCacheClearer(() => useHelpStore.getState().clear());
