import { expect, test } from "@playwright/test";
import { E2E_PASSWORD, E2E_USER_NAME } from "../../playwright.config";
import { api, signIn } from "./fixtures/api-client";
import { expectDashboard, loginThroughForm, signOut } from "./fixtures/auth";
import {
  resolvePreference,
  type Profile,
} from "../../types/profile-type";

// The profile page edits the caller's own row and preferences.
// The saved colour scheme must apply on the next load without the page
// being visited, and rows-per-page must drive the list pages.

test.describe("profile page", () => {
  test("edits persist; colour scheme and rows per page apply after reload", async ({
    page,
  }) => {
    await page.goto("/userPreferences");
    await expect(page.getByRole("heading", { name: "Profile" })).toBeVisible();
    const firstName = page.getByLabel("First name");
    await expect(firstName).not.toHaveValue("");
    const originalFirstName = await firstName.inputValue();

    await firstName.fill("Playwright");
    await page.getByRole("textbox", { name: "Colour scheme" }).click();
    await page.getByRole("option", { name: "Dark" }).click();
    const rowsPerPage = page.getByRole("textbox", { name: "Rows per page" });
    await rowsPerPage.click();
    await page.getByRole("option", { name: "10", exact: true }).click();
    await page.getByRole("button", { name: "Save" }).click();
    await expect(page.getByText("Profile and preferences saved")).toBeVisible();
    await expect(page.locator("html")).toHaveAttribute(
      "data-mantine-color-scheme",
      "dark",
    );

    // A fresh load elsewhere in the app picks the preference up from /me.
    await page.goto("/dashboard");
    await expect(page.locator("html")).toHaveAttribute(
      "data-mantine-color-scheme",
      "dark",
    );

    await page.goto("/userPreferences");
    await expect(page.getByLabel("First name")).toHaveValue("Playwright");
    await expect(rowsPerPage).toHaveValue("10");

    // Put the fixture user back.
    await page.getByLabel("First name").fill(originalFirstName);
    await page.getByRole("textbox", { name: "Colour scheme" }).click();
    await page.getByRole("option", { name: "Follow the system" }).click();
    await rowsPerPage.click();
    await page.getByRole("option", { name: "15", exact: true }).click();
    await page.getByRole("button", { name: "Save" }).click();
    await expect(page.getByText("Profile and preferences saved")).toBeVisible();
    await expect(page.locator("html")).not.toHaveAttribute(
      "data-mantine-color-scheme",
      "dark",
    );
  });
});

test.describe("colour scheme", () => {
  // Signs out at the end, so it gets its own session rather than ending the
  // saved one every other spec uses.
  test.use({ storageState: { cookies: [], origins: [] } });

  test("a stored scheme holds while /me loads; sign-out clears it", async ({
    page,
  }) => {
    const html = page.locator("html");
    try {
      await loginThroughForm(page);
      await expectDashboard(page);
      await page.goto("/userPreferences");
      await page.getByRole("textbox", { name: "Colour scheme" }).click();
      await page.getByRole("option", { name: "Dark" }).click();
      await page.getByRole("button", { name: "Save" }).click();
      await expect(
        page.getByText("Profile and preferences saved"),
      ).toBeVisible();
      await expect(html).toHaveAttribute("data-mantine-color-scheme", "dark");

      // Hold /me so the reload runs signed in with no profile yet. The old
      // code wrote the default scheme over the stored one in that gap.
      let releaseMe = () => {};
      const meHeld = new Promise<void>((resolve) => {
        releaseMe = resolve;
      });
      let meRequested = false;
      await page.route("**/api/me", async (route) => {
        if (route.request().method() !== "GET") return route.fallback();
        meRequested = true;
        await meHeld;
        await route.fallback();
      });
      await page.reload();
      await expect(
        page.getByRole("heading", { name: "Profile" }),
      ).toBeVisible();
      await expect.poll(() => meRequested).toBe(true);
      await expect(html).toHaveAttribute("data-mantine-color-scheme", "dark");
      releaseMe();
      await expect(page.getByLabel("First name")).not.toHaveValue("");
      await expect(html).toHaveAttribute("data-mantine-color-scheme", "dark");
      await page.unrouteAll({ behavior: "wait" });

      // The next person at this browser starts from the default.
      await signOut(page);
      await expect(page).toHaveURL(/\/login/);
      await expect(html).not.toHaveAttribute(
        "data-mantine-color-scheme",
        "dark",
      );
      await page.reload();
      await expect(page.getByLabel("Username")).toBeVisible();
      await expect(html).not.toHaveAttribute(
        "data-mantine-color-scheme",
        "dark",
      );
    } finally {
      // Put the fixture user back whatever failed above.
      const apiSession = await signIn(E2E_USER_NAME, E2E_PASSWORD);
      const profile = await api.get<Profile>(apiSession, "/me");
      await api.put<Profile>(apiSession, "/me", {
        firstName: profile.firstName,
        lastName: profile.lastName,
        email: profile.email,
        preferenceJsonb: {
          ...resolvePreference(profile.preferenceJsonb),
          colorScheme: "auto",
        },
        updateTs: profile.updateTs,
      });
    }
  });
});
