import { describe, expect, it } from "vitest";
import {
  GLOSSARY,
  TOPICS,
  WHATS_NEW,
  newestFirst,
  audienceFromGrants,
  getEntry,
  latestWhatsNewId,
  searchHelp,
  tourForRoute,
} from "../help/help-registry";
import { DEFAULT_HELP_AUDIENCE } from "../help/content/audience";
import { routeKeyOf } from "../help/use-route-key";
import { ROUTES } from "../constants/routes";

// The registry over the template's own content: lookups, search ranking
// and the audience default.

describe("help registry", () => {
  it("finds a term or a topic by id, and nothing else", () => {
    expect(getEntry("mfa")?.kind).toBe("term");
    expect(getEntry("adding-a-user")?.kind).toBe("topic");
    expect(getEntry("no-such-entry")).toBeUndefined();
  });

  it("ranks an exact title above a body mention", () => {
    const resultList = searchHelp("invite link");
    expect(resultList.length).toBeGreaterThan(1);
    expect(resultList[0].id).toBe("invite-link");
  });

  it("finds a term by its alias, ignoring case and accents", () => {
    expect(searchHelp("TENANT")[0]?.id).toBe("security-context");
    expect(searchHelp("2FA")[0]?.id).toBe("mfa");
    expect(searchHelp("organisätion")[0]?.id).toBe("security-context");
  });

  it("needs every word to match", () => {
    expect(searchHelp("invite zebra")).toEqual([]);
    expect(searchHelp("   ")).toEqual([]);
  });

  it("caps the result count", () => {
    expect(searchHelp("the", 2).length).toBeLessThanOrEqual(2);
    expect(searchHelp("the").length).toBeLessThanOrEqual(12);
  });

  it("searches every term and topic it holds", () => {
    for (const term of GLOSSARY) {
      expect(searchHelp(term.term).map((result) => result.id)).toContain(
        term.id,
      );
    }
    for (const topic of TOPICS) {
      expect(searchHelp(topic.title).map((result) => result.id)).toContain(
        topic.id,
      );
    }
  });

  it("is the default audience with the template's empty grant map", () => {
    expect(audienceFromGrants([])).toBe(DEFAULT_HELP_AUDIENCE);
    expect(audienceFromGrants([1, 9001])).toBe(DEFAULT_HELP_AUDIENCE);
  });

  it("orders what's new newest first, whatever the file order", () => {
    const entryOf = (id: string) => ({ id, date: "", title: "", body: "" });
    const sorted = [
      "2026-10-10-1-a",
      "2026-11-02-1-c",
      "2026-10-10-2-b",
    ].map(entryOf).sort(newestFirst);
    expect(sorted.map((entry) => entry.id)).toEqual([
      "2026-11-02-1-c",
      "2026-10-10-2-b",
      "2026-10-10-1-a",
    ]);
    expect(WHATS_NEW.length).toBeGreaterThan(0);
  });

  it("finds a page's tour and the newest news", () => {
    expect(tourForRoute(ROUTES.EXAMPLE_LIST)?.id).toBe("example-list");
    expect(tourForRoute(ROUTES.DASHBOARD)).toBeUndefined();
    expect(latestWhatsNewId()).toMatch(/^\d{4}-\d{2}-\d{2}-/);
  });

  it("keys a page by its path, the index route as the login form", () => {
    expect(routeKeyOf("/exampleList")).toBe(ROUTES.EXAMPLE_LIST);
    expect(routeKeyOf("/exampleList/")).toBe(ROUTES.EXAMPLE_LIST);
    expect(routeKeyOf("/")).toBe(ROUTES.LOGIN);
  });
});
