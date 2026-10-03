import { expect, test } from "@playwright/test";

// The entity pages are copy-paste scaffolding; these prove the two things a
// copy most easily breaks: a deep link into an edit page must load its row
// after a fresh page load, and the list page keeps its place in the
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

  // Each list page is its own copy, so each is checked.
  for (const { path, sortHeader, sortKey } of [
    { path: "/appUserList", sortHeader: /Username/, sortKey: "userName" },
    {
      path: "/appUserGroupList",
      sortHeader: /Group Name/,
      sortKey: "groupName",
    },
    { path: "/exampleList", sortHeader: /Text/, sortKey: "exampleText" },
  ]) {
    test(`${path}: a stale page shows rows; a sort keeps other params`, async ({
      page,
    }) => {
      // Past the end (a stale link, or the last row deleted) clamps to the
      // last page rather than drawing an empty table.
      await page.goto(`${path}?page=99&keep=1`);
      await expect(page.getByRole("table")).toBeVisible();
      await expect(page.locator("tbody tr").first()).toBeVisible();

      // A sort merges into the query string and starts from page one.
      await page.getByRole("columnheader", { name: sortHeader }).click();
      await expect(page).toHaveURL(new RegExp(`sort=${sortKey}`));
      await expect(page).toHaveURL(/dir=asc/);
      await expect(page).toHaveURL(/keep=1/);
      await expect(page).not.toHaveURL(/page=/);
    });
  }

  test("an unknown dir sorts ascending", async ({ page }) => {
    await page.goto("/appUserList?sort=userName&dir=sideways");
    await expect(
      page.getByRole("columnheader", { name: /Username/ }),
    ).toHaveText(/↑/);
  });

  test("the user group list sorts by name and id", async ({ page }) => {
    const nameCellList = page.locator("tbody tr td:nth-child(2)");
    const idCellList = page.locator("tbody tr td:nth-child(1)");
    await page.goto("/appUserGroupList?sort=groupName");
    await expect(nameCellList.first()).toBeVisible();
    const ascNameList = await nameCellList.allTextContents();
    const byName = (a: string, b: string) =>
      a.toLowerCase() < b.toLowerCase() ? -1 : 1;
    expect(ascNameList).toEqual([...ascNameList].sort(byName));

    await page.getByRole("columnheader", { name: /Group Name/ }).click();
    await expect(page).toHaveURL(/dir=desc/);
    await expect(nameCellList.first()).not.toHaveText(ascNameList[0]);
    const descNameList = await nameCellList.allTextContents();
    expect(descNameList).toEqual([...ascNameList].reverse());

    await page.goto("/appUserGroupList?sort=appUserGroupId&dir=desc");
    await expect(idCellList.first()).toBeVisible();
    const idList = (await idCellList.allTextContents()).map(Number);
    expect(idList).toEqual([...idList].sort((a, b) => b - a));
  });
});
