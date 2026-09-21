// The caller's own user row (GET/PUT /me) and the preferences document
// stored on it. Keys are optional so an older document keeps working;
// resolvePreference fills the defaults.

export type ColorScheme = "light" | "dark" | "auto";

export interface Preference {
  colorScheme?: ColorScheme;
  rowsPerPage?: number;
}

export const defaultPreference: Required<Preference> = {
  colorScheme: "auto",
  rowsPerPage: 15,
};

export const resolvePreference = (
  preference?: Preference,
): Required<Preference> => ({
  colorScheme: preference?.colorScheme ?? defaultPreference.colorScheme,
  rowsPerPage:
    preference?.rowsPerPage && preference.rowsPerPage > 0
      ? preference.rowsPerPage
      : defaultPreference.rowsPerPage,
});

export interface Profile {
  appUserId: number;
  userName: string;
  firstName: string;
  lastName: string;
  email: string;
  securityContext: number;
  preferenceJsonb?: Preference;
  updateTs: number;
}

// PUT /me body: what a user may change about themselves.
export interface ProfileUpdate {
  firstName: string;
  lastName: string;
  email: string;
  preferenceJsonb?: Preference;
  updateTs: number;
}
