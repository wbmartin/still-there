import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  MAX_DISMISSED,
  defaultHelpPreference,
  dismissNudge,
  markTourDone,
  preferenceWithHelp,
  resolveHelpPreference,
} from "../help/help-preference";
import { useProfileStore } from "../stores/profile-store";
import { carriedPreference, type Profile } from "../types/profile-type";

// The help module's part of the preferences document: read field by field
// like the rest of it, written without losing the profile's own keys, and
// the profile page's save carrying it through (types/profile-type.ts).

const profileOf = (preferenceJsonb: unknown, updateTs = 10): Profile => ({
  appUserId: 7,
  userName: "alice",
  firstName: "Alice",
  lastName: "Tester",
  email: "alice@example.com",
  securityContext: 2,
  preferenceJsonb,
  updateTs,
  allowMfaExemptViaApi: false,
  issueReportsEnabled: false,
  helpEnabled: true,
});

describe("resolveHelpPreference", () => {
  it("is the default for a document with no help part", () => {
    expect(resolveHelpPreference(null)).toEqual(defaultHelpPreference);
    expect(resolveHelpPreference({ colorScheme: "dark" })).toEqual(
      defaultHelpPreference,
    );
    expect(resolveHelpPreference({ help: [1] })).toEqual(
      defaultHelpPreference,
    );
  });

  it("keeps good values and drops bad ones field by field", () => {
    expect(
      resolveHelpPreference({
        help: {
          whatsNewSeen: "2026-10-10-1-help",
          toursDone: ["example-list", "Bad Id", 3, "example-list"],
          dismissed: "not-a-list",
          extra: true,
        },
      }),
    ).toEqual({
      whatsNewSeen: "2026-10-10-1-help",
      toursDone: ["example-list"],
      dismissed: [],
    });
    expect(
      resolveHelpPreference({ help: { whatsNewSeen: "<b>" } }).whatsNewSeen,
    ).toBe("");
  });

  it("keeps only the newest ids past the cap", () => {
    const idList = Array.from({ length: MAX_DISMISSED + 5 }, (_, at) =>
      `tip-${at}`,
    );
    const resolved = resolveHelpPreference({ help: { dismissed: idList } });
    expect(resolved.dismissed.length).toBe(MAX_DISMISSED);
    expect(resolved.dismissed.at(-1)).toBe(`tip-${MAX_DISMISSED + 4}`);
  });
});

describe("one preferences document, two writers", () => {
  it("writes the help part beside the profile's own keys", () => {
    expect(
      preferenceWithHelp({ colorScheme: "dark", rowsPerPage: 50 }, (help) => ({
        ...help,
        whatsNewSeen: "2026-10-10-1-help",
      })),
    ).toEqual({
      colorScheme: "dark",
      rowsPerPage: 50,
      help: { whatsNewSeen: "2026-10-10-1-help", toursDone: [], dismissed: [] },
    });
  });

  it("carries the help part through the profile page's save", () => {
    const help = { whatsNewSeen: "x", toursDone: [], dismissed: [] };
    expect(carriedPreference({ colorScheme: "dark", help })).toEqual({ help });
    expect(carriedPreference({ colorScheme: "dark" })).toEqual({});
    expect(carriedPreference(null)).toEqual({});
  });
});

describe("saving the help preference", () => {
  let bodyList: unknown[];

  beforeEach(() => {
    bodyList = [];
    useProfileStore.setState({ profile: profileOf({ rowsPerPage: 25 }) });
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    useProfileStore.getState().clear();
  });

  const answer = (status: number, body: unknown) =>
    new Response(JSON.stringify(body), {
      status,
      headers: { "Content-Type": "application/json" },
    });

  it("PUTs /me with the change and keeps the answer", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn((_url: string, init: RequestInit) => {
        const sent = JSON.parse(init.body as string) as {
          preferenceJsonb: unknown;
        };
        bodyList.push(sent);
        return Promise.resolve(
          answer(200, { data: profileOf(sent.preferenceJsonb, 11) }),
        );
      }),
    );
    await markTourDone("example-list");
    expect(bodyList).toEqual([
      {
        firstName: "Alice",
        lastName: "Tester",
        email: "alice@example.com",
        preferenceJsonb: {
          colorScheme: "auto",
          rowsPerPage: 25,
          help: { whatsNewSeen: "", toursDone: ["example-list"], dismissed: [] },
        },
        updateTs: 10,
      },
    ]);
    const profile = useProfileStore.getState().profile;
    expect(profile?.updateTs).toBe(11);
    expect(resolveHelpPreference(profile?.preferenceJsonb).toursDone).toEqual(
      ["example-list"],
    );
  });

  it("re-reads the row after a 409 and applies the change to it", async () => {
    // Another tab dismissed a tip meanwhile; this save keeps that.
    const newer = profileOf(
      { rowsPerPage: 50, help: { dismissed: ["other-tip"] } },
      12,
    );
    const fetchMock = vi.fn((url: string, init?: RequestInit) => {
      if (init?.method === "PUT") {
        const sent = JSON.parse(init.body as string) as {
          preferenceJsonb: unknown;
          updateTs: number;
        };
        bodyList.push(sent);
        if (sent.updateTs === 10) {
          return Promise.resolve(answer(409, { error: "stale" }));
        }
        return Promise.resolve(
          answer(200, { data: profileOf(sent.preferenceJsonb, 13) }),
        );
      }
      expect(url).toBe("/api/me");
      return Promise.resolve(answer(200, { data: newer }));
    });
    vi.stubGlobal("fetch", fetchMock);

    await dismissNudge("this-tip");
    expect(bodyList).toHaveLength(2);
    expect(bodyList[1]).toMatchObject({
      updateTs: 12,
      preferenceJsonb: {
        rowsPerPage: 50,
        help: { dismissed: ["other-tip", "this-tip"] },
      },
    });
    expect(useProfileStore.getState().profile?.updateTs).toBe(13);
  });

  it("reports a failure and leaves the profile as it was", async () => {
    vi.stubGlobal("console", { ...console, error: vi.fn() });
    vi.stubGlobal(
      "fetch",
      vi.fn(() => Promise.resolve(answer(400, { error: "email is not valid" }))),
    );
    await expect(markTourDone("example-list")).resolves.toBeUndefined();
    expect(useProfileStore.getState().profile?.updateTs).toBe(10);
  });
});
