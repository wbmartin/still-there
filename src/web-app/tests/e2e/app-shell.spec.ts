import { expect, test } from "@playwright/test";

// The shell keeps navigation reachable however the page scrolls: the
// header and navbar are fixed, the nav's link list scrolls on its own, the
// page's title block pins under the header, and on a phone the nav is a
// drawer that closes once a link is tapped.

test.describe("app shell", () => {
  test("nav and main content scroll independently", async ({ page }) => {
    // Shorter than the template's five links, so the nav has to scroll.
    await page.setViewportSize({ width: 1280, height: 200 });
    await page.goto("/exampleList");
    await expect(
      page.getByRole("heading", { name: "Examples" }),
    ).toBeVisible();

    const nav = page.getByRole("navigation");
    const navViewport = nav.locator(".mantine-ScrollArea-viewport");
    const firstLink = nav.getByRole("link", { name: "Dashboard" });
    const lastLink = nav.getByRole("link", { name: "Profile" });
    const windowScrollY = () => page.evaluate(() => window.scrollY);

    await expect(firstLink).toBeInViewport();
    await expect(lastLink).not.toBeInViewport();
    expect(
      await navViewport.evaluate((el) => el.scrollHeight > el.clientHeight),
    ).toBe(true);

    // Scrolling the nav to its last link leaves the page where it was.
    await navViewport.evaluate((el) => el.scrollTo(0, el.scrollHeight));
    await expect(lastLink).toBeInViewport();
    expect(await windowScrollY()).toBe(0);

    // Scrolling the page leaves the nav where it was.
    await navViewport.evaluate((el) => el.scrollTo(0, 0));
    await page.mouse.move(800, 150);
    await page.mouse.wheel(0, 600);
    await expect.poll(windowScrollY).toBeGreaterThan(0);
    await expect(firstLink).toBeInViewport();
    expect(await navViewport.evaluate((el) => el.scrollTop)).toBe(0);
  });

  test("the page title block stays pinned under the header", async ({
    page,
  }) => {
    await page.setViewportSize({ width: 1280, height: 260 });
    await page.goto("/exampleList");
    const title = page.getByRole("heading", { name: "Examples" });
    const addButton = page.getByRole("button", { name: "Add Example" });
    const titleBlock = page.locator(".app-page > * > :has(h1)");
    const table = page.locator("table");
    await expect(title).toBeVisible();
    await expect(table.locator("tbody tr").first()).toBeVisible();

    // The hook measured the block for StickyFilters.
    await expect
      .poll(() =>
        page.locator(".app-page").evaluate((el) =>
          parseFloat(el.style.getPropertyValue("--app-page-header-height")),
        ),
      )
      .toBeGreaterThan(0);

    const top = async (locator: typeof title) =>
      (await locator.boundingBox())?.y ?? Number.NaN;
    const blockTopBefore = await top(titleBlock);
    const tableTopBefore = await top(table);

    await page.evaluate(() =>
      window.scrollTo(0, document.documentElement.scrollHeight),
    );
    const scrollY = await page.evaluate(() => window.scrollY);
    // Far enough that an unpinned block would be under the 56 px header.
    expect(scrollY).toBeGreaterThan(blockTopBefore - 56 + 20);

    // The table moved with the page; the block stopped at the header's
    // bottom edge, with the title and its button still on screen.
    expect(await top(table)).toBeCloseTo(tableTopBefore - scrollY, 0);
    expect(await top(titleBlock)).toBeCloseTo(56, 0);
    await expect(title).toBeInViewport();
    await expect(addButton).toBeInViewport();
  });

  test("on a phone the burger opens the nav and a tap closes it", async ({
    page,
  }) => {
    await page.setViewportSize({ width: 390, height: 800 });
    await page.goto("/dashboard");
    await expect(
      page.getByRole("heading", { name: "Dashboard" }),
    ).toBeVisible();

    const examplesLink = page
      .getByRole("navigation")
      .getByRole("link", { name: "Examples" });
    await expect(examplesLink).not.toBeInViewport();

    await page.getByRole("button", { name: "Toggle navigation" }).click();
    await expect(examplesLink).toBeInViewport();

    await examplesLink.click();
    await expect(page).toHaveURL(/\/exampleList/);
    await expect(
      page.getByRole("heading", { name: "Examples" }),
    ).toBeVisible();
    await expect(examplesLink).not.toBeInViewport();
  });
});
