// What each user has seen and finished in the help module, kept under
// `help` in their own preferences document (app_user.preference_jsonb,
// GET/PUT /me) so it follows them from browser to browser
// (doc/151-help-module.md). The rest of the document is
// types/profile-type.ts's; each writer carries the other's keys through.
import { API_ENDPOINTS } from "../constants/api";
import { useProfileStore } from "../stores/profile-store";
import type { ApiResponse } from "../types/api";
import { resolvePreference, type Profile } from "../types/profile-type";
import { apiJson, isApiStatus, jsonBody } from "../utils/api";
import { reportError } from "../utils/report-error";
import { latestWhatsNewId } from "./help-registry";

export interface HelpPreference {
  // The newest "What's new" id the user has seen; "" for none.
  whatsNewSeen: string;
  // Tours finished or skipped.
  toursDone: string[];
  // Nudges dismissed.
  dismissed: string[];
}

export const defaultHelpPreference: HelpPreference = {
  whatsNewSeen: "",
  toursDone: [],
  dismissed: [],
};

// Bounds that keep the document well under the API's 8 KiB: the newest
// ids are kept.
export const MAX_TOURS_DONE = 50;
export const MAX_DISMISSED = 100;

const HELP_ID_PATTERN = /^[a-z0-9][a-z0-9-]{0,63}$/;

const idListOf = (value: unknown, max: number): string[] =>
  Array.isArray(value)
    ? [
        ...new Set(
          value.filter(
            (id): id is string =>
              typeof id === "string" && HELP_ID_PATTERN.test(id),
          ),
        ),
      ].slice(-max)
    : [];

// Field by field, like resolvePreference: an unknown key is ignored, a bad
// value is that field's default. Never throws.
export function resolveHelpPreference(stored: unknown): HelpPreference {
  if (typeof stored !== "object" || stored === null || Array.isArray(stored)) {
    return defaultHelpPreference;
  }
  const help = (stored as { help?: unknown }).help;
  if (typeof help !== "object" || help === null || Array.isArray(help)) {
    return defaultHelpPreference;
  }
  const field = help as {
    whatsNewSeen?: unknown;
    toursDone?: unknown;
    dismissed?: unknown;
  };
  return {
    whatsNewSeen:
      typeof field.whatsNewSeen === "string" &&
      (field.whatsNewSeen === "" || HELP_ID_PATTERN.test(field.whatsNewSeen))
        ? field.whatsNewSeen
        : "",
    toursDone: idListOf(field.toursDone, MAX_TOURS_DONE),
    dismissed: idListOf(field.dismissed, MAX_DISMISSED),
  };
}

// The document PUT /me sends: the profile's own keys as they resolve, the
// help part with the change applied.
export function preferenceWithHelp(
  stored: unknown,
  change: (help: HelpPreference) => HelpPreference,
) {
  return {
    ...resolvePreference(stored),
    help: change(resolveHelpPreference(stored)),
  };
}

const putPreference = async (
  profile: Profile,
  change: (help: HelpPreference) => HelpPreference,
): Promise<Profile> => {
  const response = await apiJson<ApiResponse<Profile>>(API_ENDPOINTS.ME, {
    method: "PUT",
    ...jsonBody({
      firstName: profile.firstName,
      lastName: profile.lastName,
      email: profile.email,
      preferenceJsonb: preferenceWithHelp(profile.preferenceJsonb, change),
      updateTs: profile.updateTs,
    }),
  });
  return response.data;
};

/**
 * Saves a change to the signed-in user's help preference. The change is a
 * function, so a retry after a 409 (the row changed: another tab, the
 * Profile page) applies it to the row as it now stands. Best-effort: a
 * failure is reported, never thrown, and the help keeps working.
 */
export async function saveHelpPreference(
  change: (help: HelpPreference) => HelpPreference,
): Promise<void> {
  const profile = useProfileStore.getState().profile;
  if (!profile) return;
  try {
    let saved: Profile;
    try {
      saved = await putPreference(profile, change);
    } catch (err) {
      if (!isApiStatus(err, 409)) throw err;
      const current = await useProfileStore.getState().readProfile();
      saved = await putPreference(current, change);
    }
    useProfileStore.setState({ profile: saved });
  } catch (err) {
    reportError(err, "saving help preference");
  }
}

export const markWhatsNewSeen = (id: string) =>
  saveHelpPreference((help) => ({ ...help, whatsNewSeen: id }));

export const markTourDone = (tourId: string) =>
  saveHelpPreference((help) => ({
    ...help,
    toursDone: [...help.toursDone.filter((id) => id !== tourId), tourId]
      .slice(-MAX_TOURS_DONE),
  }));

export const dismissNudge = (nudgeId: string) =>
  saveHelpPreference((help) => ({
    ...help,
    dismissed: [...help.dismissed.filter((id) => id !== nudgeId), nudgeId]
      .slice(-MAX_DISMISSED),
  }));

/** The server's HELP_ENABLED, from /me; false until the profile loads. */
export const useHelpEnabled = (): boolean =>
  useProfileStore((state) => state.profile?.helpEnabled === true);

/**
 * The signed-in user's help preference and whether it has loaded: until
 * it has, a nudge or a first-visit tour waits rather than flash.
 */
export function useHelpPreference(): {
  preference: HelpPreference;
  loaded: boolean;
} {
  const stored = useProfileStore((state) => state.profile?.preferenceJsonb);
  const loaded = useProfileStore((state) => state.profile !== null);
  return { preference: resolveHelpPreference(stored), loaded };
}

/** True while there is a "What's new" entry newer than the user has seen. */
export function useWhatsNewUnread(): boolean {
  const { preference, loaded } = useHelpPreference();
  const latest = latestWhatsNewId();
  return loaded && latest !== undefined && preference.whatsNewSeen < latest;
}
