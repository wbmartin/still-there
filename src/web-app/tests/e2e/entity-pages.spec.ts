import { expect, test } from "@playwright/test";

// The entity pages are copy-paste scaffolding; these prove the two things a
// copy most easily breaks: a deep link into an edit page must load its row
// after a fresh page load (PS0012), and the list page keeps its place in the
// URL.

test.describe("entity pages", () => {
  test("a deep link to a user group loads the group after a refresh", async ({
    page,
  }) => {
    // Group 2 comes from the fixtures. Nothing in the store yet: this is a
    // cold load, the case that used to yield an empty form.
    await page.goto("/appUserGroupEdit?appUserGroupId=2");
    await expect(page.getByLabel("Group Name")).toHaveValue("Admin");

    await page.reload();
    await expect(page.getByLabel("Group Name")).toHaveValue("Admin");
  });

  test("the list page and sort live in the URL", async ({ page }) => {
    await page.goto("/appUserList");
    await page.getByRole("columnheader", { name: /Username/ }).click();
    await expect(page).toHaveURL(/sort=userName/);
    await expect(page).toHaveURL(/dir=asc/);
  });
});
