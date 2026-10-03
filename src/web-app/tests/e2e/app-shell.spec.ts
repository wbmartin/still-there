import { expect, test } from "@playwright/test";

// The shell keeps navigation reachable however the page scrolls: the
// header and navbar are fixed, the nav's link list scrolls on its own, and
// on a phone the nav is a drawer that closes once a link is tapped.

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
