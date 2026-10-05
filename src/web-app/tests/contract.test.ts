import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { describe, expect, it } from "vitest";
import { API_ENDPOINTS } from "../constants/api";
import { grants } from "../constants/grants";

// The web app's Go mirrors are hand-coded (no generator) and drift-tested
// here. Grants: Go writes src/api-srv/internal/model/testdata/grants.json
// from AllKnownGrants (TestGrantsFixtureIsCurrent, refreshed only by
// bin/305-update-grants-fixture); this file replays it against
// constants/grants.ts both ways. Routes: every API_ENDPOINTS path must be
// registered in src/api-srv/cmd/server/main.go. Go routes the web never calls
// (/health, test-only routes) are allowed.

// Paths are relative to src/web-app.
function readRepoFile(pathFromWebApp: string): string {
  const absolutePath = resolve(import.meta.dirname, "..", pathFromWebApp);
  return readFileSync(absolutePath, "utf8");
}

const goGrantMap = JSON.parse(
  readRepoFile("../api-srv/internal/model/testdata/grants.json"),
) as Record<string, number>;

// grants also carries helper methods; the grant ids are its number values.
const webGrantMap: Record<string, number> = {};
for (const [shortCode, value] of Object.entries(grants)) {
  if (typeof value === "number") webGrantMap[shortCode] = value;
}

describe("grant mirror: constants/grants.ts vs model/grants.go", () => {
  it("reads a non-trivial fixture (floor)", () => {
    expect(Object.keys(goGrantMap).length).toBeGreaterThanOrEqual(18);
  });

  it("has every Go grant, with the same id", () => {
    for (const [shortCode, id] of Object.entries(goGrantMap)) {
      expect(webGrantMap[shortCode], shortCode).toBe(id);
    }
  });

  it("has no grant Go does not know", () => {
    const extraList = Object.keys(webGrantMap).filter(
      (shortCode) => !(shortCode in goGrantMap),
    );
    expect(extraList).toEqual([]);
  });
});

// `mux.HandleFunc("GET /path", ...)`, the pattern possibly on the next line.
const routePattern = /mux\.HandleFunc\(\s*"([A-Z]+) (\/[^"\s]*)"/g;

const goRouteList = [
  ...readRepoFile("../api-srv/cmd/server/main.go").matchAll(routePattern),
].map((match) => ({ method: match[1], path: match[2] }));

describe("route mirror: constants/api.ts vs cmd/server/main.go", () => {
  it("parses the Go routes (canary)", () => {
    expect(goRouteList.length).toBeGreaterThanOrEqual(25);
  });

  it("registers every path the web app calls", () => {
    const goPathSet = new Set(goRouteList.map((route) => route.path));
    const missingList = Object.entries(API_ENDPOINTS)
      .filter(([, path]) => !goPathSet.has(path))
      .map(([name, path]) => `${name}: ${path}`);
    expect(missingList).toEqual([]);
  });
});
