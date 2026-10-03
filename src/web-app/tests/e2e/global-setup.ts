import { execFileSync } from "node:child_process";
import { mkdirSync, writeFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { E2E_PASSWORD, E2E_USER_NAME } from "../../playwright.config";
import { cookieForStorageState, signIn } from "./fixtures/api-client";

// Runs once, before any spec.
//
//   1. Put the e2e database back to its seeded fixtures.
//   2. Sign in ONCE and save the auth cookie, so every spec starts already
//      authenticated. Signing in is a behaviour ONE spec should own;
//      everywhere else it is setup.
//
// The auth cookie is httpOnly, so nothing in the app can read it back -- but
// that does not matter here, because the app calls /refresh on load, and that
// is what re-establishes the CSRF token in memory. Restoring the cookie alone
// is therefore enough to restore a working session.

// The package is ESM, so there is no __dirname. Derived from import.meta.url
// rather than from process.cwd(), which Playwright does not guarantee.
const HERE = dirname(fileURLToPath(import.meta.url));
const STATE_PATH = resolve(HERE, ".auth/state.json");
const SESSION_PATH = resolve(HERE, ".auth/session.json");

// The tenant the fixture logins live in (sql/fixtures/test-seed.sql). Tenant
// 1 is the first real tenant and does not exist in a test database.
const TEST_TENANT = 2;

export default async function globalSetup() {
  // Reset the database first, every run, without being asked. Resetting here
  // rather than in the wrapper script is what makes a bare
  // `npx playwright test` reproducible.
  //
  // TRUNCATE + re-seed, not DROP. The API server is already running by the
  // time this executes, so dropping the database leaves every connection in
  // its pgx pool pointing at something that no longer exists -- the app then
  // renders "Cannot reach the server" and the suite fails for a reason that
  // has nothing to do with the code under test. See bin/515.
  // bin/300-run-tests has just rebuilt the database when it sets
  // E2E_DB_SEEDED; reseeding again would only cost time.
  if (!process.env.E2E_DB_SEEDED) {
    execFileSync(resolve(HERE, "../../../../bin/515-reseed-db"), {
      stdio: "inherit",
    });
  }

  const session = await signIn(E2E_USER_NAME, E2E_PASSWORD);
  // A login outside the test tenant means this API serves a database that
  // is not a test database: stop before any spec writes to it.
  if (session.securityContext !== TEST_TENANT) {
    throw new Error(
      `[e2e] ${E2E_USER_NAME} signed in to tenant ` +
        `${session.securityContext}, not ${TEST_TENANT}: refusing to run ` +
        `against a database that is not a test database`,
    );
  }

  mkdirSync(dirname(STATE_PATH), { recursive: true });
  writeFileSync(
    STATE_PATH,
    JSON.stringify(
      { cookies: [cookieForStorageState(session)], origins: [] },
      null,
      2,
    ),
  );
  // For specs that build rows through the API. Written to disk rather than
  // passed through a module global because Playwright runs global setup in a
  // different process from the workers.
  writeFileSync(SESSION_PATH, JSON.stringify(session, null, 2));

  console.log(
    `[e2e] signed in as ${E2E_USER_NAME} ` +
      `(security context ${session.securityContext})`,
  );
}
