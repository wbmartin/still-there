// The caller's own user row (GET/PUT /me) and the preferences document
// stored on it. The API checks only that the document is a JSON object of
// at most 8 KiB, so the keys are this file's contract and a stored value
// can be anything (an older build's key, a direct PUT /me, a hand-edited
// row). Nothing reads the raw document: resolvePreference is the way in,
// and it answers with a complete, valid Preference whatever it is handed.

// The Mantine colour schemes offered; "auto" follows the system.
export const colorSchemeList = ["auto", "light", "dark"] as const;

export type ColorScheme = (typeof colorSchemeList)[number];

// The page sizes offered for the list pages. A fixed list, not a range:
// a stored 100000 would otherwise render every row of every list.
export const rowsPerPageList = [10, 15, 25, 50, 100] as const;

export type RowsPerPage = (typeof rowsPerPageList)[number];

export interface Preference {
  colorScheme: ColorScheme;
  rowsPerPage: RowsPerPage;
}

export const defaultPreference: Preference = {
  colorScheme: "auto",
  rowsPerPage: 15,
};

export const isColorScheme = (value: unknown): value is ColorScheme =>
  typeof value === "string" &&
  (colorSchemeList as readonly string[]).includes(value);

export const isRowsPerPage = (value: unknown): value is RowsPerPage =>
  typeof value === "number" &&
  (rowsPerPageList as readonly number[]).includes(value);

// Field by field: an unknown key is ignored, an invalid value falls back
// to that field's default alone, and null / a non-object / an array is the
// whole default. Never throws, never returns a partial object.
export const resolvePreference = (stored: unknown): Preference => {
  if (typeof stored !== "object" || stored === null || Array.isArray(stored)) {
    return defaultPreference;
  }
  const field = stored as { colorScheme?: unknown; rowsPerPage?: unknown };
  return {
    colorScheme: isColorScheme(field.colorScheme)
      ? field.colorScheme
      : defaultPreference.colorScheme,
    rowsPerPage: isRowsPerPage(field.rowsPerPage)
      ? field.rowsPerPage
      : defaultPreference.rowsPerPage,
  };
};

export interface Profile {
  appUserId: number;
  userName: string;
  firstName: string;
  lastName: string;
  email: string;
  securityContext: number;
  // As stored; read it through resolvePreference.
  preferenceJsonb?: unknown;
  updateTs: number;
  // The server's ALLOW_MFA_EXEMPT_VIA_API (B097): false means the API
  // refuses setting mfaExempt, so the user edit page hides the checkbox.
  allowMfaExemptViaApi: boolean;
}

// PUT /me body: what a user may change about themselves.
export interface ProfileUpdate {
  firstName: string;
  lastName: string;
  email: string;
  preferenceJsonb?: Preference;
  updateTs: number;
}
