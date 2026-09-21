import { expect, test } from "@playwright/test";

// The profile page edits the caller's own row and preferences (PS0018).
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
    await page.getByLabel("Rows per page").fill("5");
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
    await expect(page.getByLabel("Rows per page")).toHaveValue("5");

    // Put the fixture user back.
    await page.getByLabel("First name").fill(originalFirstName);
    await page.getByRole("textbox", { name: "Colour scheme" }).click();
    await page.getByRole("option", { name: "Follow the system" }).click();
    await page.getByLabel("Rows per page").fill("15");
    await page.getByRole("button", { name: "Save" }).click();
    await expect(page.getByText("Profile and preferences saved")).toBeVisible();
    await expect(page.locator("html")).not.toHaveAttribute(
      "data-mantine-color-scheme",
      "dark",
    );
  });
});
