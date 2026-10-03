import { describe, expect, it } from "vitest";
import { defaultPreference, resolvePreference } from "../types/profile-type";

// B062: resolvePreference is the only reader of app_user.preference_jsonb,
// and the column holds whatever was stored there (an older build's keys, a
// direct PUT /me, a hand-edited row). These pin what each of those means.

describe("resolvePreference", () => {
  it("resolves a complete stored object as it stands", () => {
    expect(resolvePreference({ colorScheme: "dark", rowsPerPage: 50 })).toEqual(
      { colorScheme: "dark", rowsPerPage: 50 },
    );
  });

  it("is the whole default for a login with nothing saved", () => {
    expect(resolvePreference(null)).toEqual(defaultPreference);
    expect(resolvePreference(undefined)).toEqual(defaultPreference);
  });

  it("is the whole default for a stored value that is not an object", () => {
    expect(resolvePreference("dark")).toEqual(defaultPreference);
    expect(resolvePreference([{ colorScheme: "dark" }])).toEqual(
      defaultPreference,
    );
  });

  it("falls back per field, so one bad value does not lose the other", () => {
    expect(
      resolvePreference({ colorScheme: "purple", rowsPerPage: 50 }),
    ).toEqual({ colorScheme: defaultPreference.colorScheme, rowsPerPage: 50 });
    expect(
      resolvePreference({ colorScheme: "dark", rowsPerPage: 100000 }),
    ).toEqual({ colorScheme: "dark", rowsPerPage: defaultPreference.rowsPerPage });
  });

  it("refuses a page size outside the offered list, whatever its type", () => {
    // 37 would work as a number; it is still not an option, and a value
    // nobody chose in the form is one the form cannot show back.
    expect(resolvePreference({ rowsPerPage: 37 }).rowsPerPage).toBe(
      defaultPreference.rowsPerPage,
    );
    expect(resolvePreference({ rowsPerPage: "50" }).rowsPerPage).toBe(
      defaultPreference.rowsPerPage,
    );
  });

  it("ignores keys it does not know and keeps the ones it does", () => {
    expect(
      resolvePreference({ colorScheme: "light", density: "compact" }),
    ).toEqual({ colorScheme: "light", rowsPerPage: defaultPreference.rowsPerPage });
  });
});
