import { expect, test } from "@playwright/test";
import { expectDashboard, loginThroughForm } from "./fixtures/auth";

// Proves the harness itself before anything is built on it: the stack
// starts, the fixtures exist, the saved session is accepted, and the login
// screen works end to end. If these fail, every later spec fails too, and a
// suite should say "the server did not start" in one line rather than twenty.

test.describe("harness", () => {
  test("the seeded session is already signed in", async ({ page }) => {
    // No login step. If the saved cookie were not accepted, the auth guard
    // would redirect to /login and the dashboard would never render.
    await page.goto("/dashboard");
    await expectDashboard(page);
  });
});

test.describe("signing in and out", () => {
  // The only tests in the suite that start SIGNED OUT.
  test.use({ storageState: { cookies: [], origins: [] } });

  test("login reaches the dashboard, examples, then logout", async ({
    page,
  }) => {
    await loginThroughForm(page);
    await expectDashboard(page);
    // Two example rows come from the fixtures, so the count is not a guess.
    await expect(page.getByText("Total Examples: 2")).toBeVisible();

    await page.getByRole("link", { name: "Examples" }).click();
    await expect(page).toHaveURL(/\/exampleList/);
    await expect(
      page.getByRole("heading", { name: "Examples" }),
    ).toBeVisible();

    // Logout must end the session, not just the view: a protected URL
    // afterwards has to bounce back to the form.
    await page.getByRole("button", { name: "Logout" }).click();
    await expect(page).toHaveURL(/\/login/);
    await page.goto("/dashboard");
    await expect(page).toHaveURL(/\/login/);
    await expect(page.getByLabel("Username")).toBeVisible();
  });

  test("a wrong password says so and stays on the form", async ({ page }) => {
    await loginThroughForm(page, undefined, "definitely-not-the-password");
    await expect(
      page.getByText("Login Failed", { exact: true }),
    ).toBeVisible();
    await expect(page).toHaveURL(/\/login/);
  });
});
