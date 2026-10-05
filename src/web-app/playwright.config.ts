import { defineConfig, devices } from "@playwright/test";
import { readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

// End-to-end tests for the React app.
//
// Ports are deliberately NOT the dev stack's. The ordinary development
// servers on 5173/8080 keep running untouched while this suite drives its own
// pair, so a test run never fights whatever is open in a browser tab -- and
// never writes to the database the developer is looking at.
//
// The API's config file is the one source for its port and bind address:
// E2E_API_ENV (bin/300 exports it from etc/loc-test.env) or the repo's
// etc/e2e-api-srv.env. Read here so the two cannot disagree.
const HERE = dirname(fileURLToPath(import.meta.url));
export const E2E_API_ENV_PATH = resolve(
  HERE,
  process.env.E2E_API_ENV ?? "../../etc/e2e-api-srv.env",
);
const apiEnv = Object.fromEntries(
  readFileSync(E2E_API_ENV_PATH, "utf8")
    .split("\n")
    .map((l) => l.trim())
    .filter((l) => l && !l.startsWith("#") && l.includes("="))
    .map((l) => {
      const i = l.indexOf("=");
      return [l.slice(0, i).trim(), l.slice(i + 1).trim()];
    }),
);
if (!apiEnv.SERVER_PORT) {
  throw new Error(`SERVER_PORT missing from ${E2E_API_ENV_PATH}`);
}
export const E2E_WEB_PORT = 5182;
export const E2E_API_PORT = Number(apiEnv.SERVER_PORT);
export const E2E_BASE_URL = `http://localhost:${E2E_WEB_PORT}`;
export const E2E_API_ROOT = `http://${apiEnv.SERVER_BIND_ADDRESS || "127.0.0.1"}:${E2E_API_PORT}`;

// One place, so a spec and the seed cannot disagree about who is signing in.
// The defaults are the fixture user bin/510 seeds (tenant 2) from
// src/api-srv/sql/fixtures/test-seed.sql; a host whose fixtures differ
// overrides them in the environment.
export const E2E_USER_NAME = process.env.E2E_USER_NAME ?? "admin";
export const E2E_PASSWORD = process.env.E2E_PASSWORD ?? "admin123!";

export default defineConfig({
  testDir: "./tests/e2e",
  // Vitest owns tests/**/*.test.ts (vitest.config.ts); Playwright owns
  // *.spec.ts. Matching only *.spec.ts keeps the two frameworks in one
  // directory tree without either trying to execute the other's files.
  testMatch: "**/*.spec.ts",

  fullyParallel: false,
  // Serial: the fixtures are shared rows in one security context, so two
  // workers editing the same example would fail each other rather than the
  // code.
  workers: 1,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  reporter: process.env.CI ? "list" : [["list"], ["html", { open: "never" }]],

  use: {
    baseURL: E2E_BASE_URL,
    trace: "on-first-retry",
    screenshot: "only-on-failure",
    // Signed in once in global setup and reused by every spec. Signing in is
    // a behaviour ONE spec should own; everywhere else it is setup.
    storageState: "tests/e2e/.auth/state.json",
  },

  globalSetup: "./tests/e2e/global-setup.ts",

  projects: [{ name: "chromium", use: { ...devices["Desktop Chrome"] } }],

  // The suite starts its own stack, so `npx playwright test` is
  // self-contained rather than testing whatever happens to be listening.
  webServer: [
    {
      // The API's config file (see E2E_API_ENV_PATH above; bin/230 makes
      // a project-relative E2E_API_ENV absolute first).
      command: `go run ./cmd/server ${E2E_API_ENV_PATH}`,
      cwd: "../api-srv",
      url: `${E2E_API_ROOT}/health`,
      reuseExistingServer: !process.env.CI,
      stdout: "pipe",
      stderr: "pipe",
      timeout: 120_000,
    },
    {
      // vite.config.ts reads VITE_API_PROXY_URL through loadEnv, which
      // includes the process environment, so no .env file is needed.
      command: `npx vite --port ${E2E_WEB_PORT} --strictPort`,
      url: E2E_BASE_URL,
      reuseExistingServer: !process.env.CI,
      env: { VITE_API_PROXY_URL: E2E_API_ROOT },
      timeout: 60_000,
    },
  ],
});
