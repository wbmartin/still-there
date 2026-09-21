import { readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { expect, type Page } from "@playwright/test";
import { E2E_PASSWORD, E2E_USER_NAME } from "../../../playwright.config";
import type { ApiSession } from "./api-client";

// The signed-in state every spec starts from, and the one way to get there
// by hand for the spec that owns the login screen.

// The package is ESM, so there is no __dirname.
const HERE = dirname(fileURLToPath(import.meta.url));

/**
 * The API session global setup signed in with, for a spec that needs to
 * build rows through the API. Read from disk because Playwright runs global
 * setup in a different process from the workers.
 */
export const session = (): ApiSession =>
  JSON.parse(
    readFileSync(resolve(HERE, "../.auth/session.json"), "utf8"),
  ) as ApiSession;

/** Drive the login form as the seeded fixture user. */
export const loginThroughForm = async (
  page: Page,
  userName = E2E_USER_NAME,
  password = E2E_PASSWORD,
): Promise<void> => {
  await page.goto("/login");
  await page.getByLabel("Username").fill(userName);
  await page.getByLabel("Password").fill(password);
  await page.getByRole("button", { name: "Login" }).click();
};

/** Wait for the dashboard to have loaded its data, not just its route. */
export const expectDashboard = async (page: Page): Promise<void> => {
  await expect(page).toHaveURL(/\/dashboard/);
  await expect(
    page.getByRole("heading", { name: "Dashboard" }),
  ).toBeVisible();
  await expect(page.getByText(/Total Examples: \d+/)).toBeVisible();
};
