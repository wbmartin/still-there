import { expect, test } from "@playwright/test";

// The issue-report module, on in the e2e API (etc/e2e-api-srv.env): the
// header's Report button opens the drawer, a report is sent with a capture
// of the page, and it opens on the Issue reports page with its picture.
// The relay is off there, so the report stays in the app ("Received").

test.describe("issue reports", () => {
  test("report from the header, with a screen capture", async ({ page }) => {
    await page.goto("/dashboard");
    await expect(
      page.getByRole("heading", { name: "Dashboard" }),
    ).toBeVisible();

    await page.getByRole("button", { name: "Report", exact: true }).click();
    const drawer = page.getByRole("dialog", { name: "Report an issue" });
    await expect(drawer).toBeVisible();
    const summary = `Playwright report ${Date.now()}`;
    await drawer.getByRole("textbox", { name: "Summary" }).fill(summary);
    await drawer
      .getByRole("textbox", { name: "Description" })
      .fill("Steps:\n\n1. Open the **dashboard**");
    await drawer.getByRole("button", { name: "Capture screen" }).click();
    await expect(drawer.getByText(/^screenshot-.*\.png$/)).toBeVisible();
    await drawer.getByRole("button", { name: "Send report" }).click();

    await expect(page).toHaveURL(/\/issueReportList\?issueReportId=\d+/);
    await expect(
      page.getByRole("heading", { name: "Issue reports" }),
    ).toBeVisible();
    await expect(page.getByRole("heading", { name: summary })).toBeVisible();
    await expect(page.getByText("Received").first()).toBeVisible();
    // The markdown renders; the capture is shown from the attachment route.
    await expect(page.locator("strong", { hasText: "dashboard" }))
      .toBeVisible();
    const picture = page.locator("img[alt^='screenshot-']");
    await expect(picture).toBeVisible();
    await expect
      .poll(() =>
        picture.evaluate((image: HTMLImageElement) => image.naturalWidth),
      )
      .toBeGreaterThan(0);

    // The reporter withdraws it; it moves to Closed.
    await page.getByRole("button", { name: "Withdraw" }).click();
    const confirm = page.getByRole("dialog", { name: "Withdraw this report" });
    await confirm.getByRole("button", { name: "Withdraw" }).click();
    await expect(confirm).toBeHidden();
    await expect(page.getByText("Withdrawn").first()).toBeVisible();
    await expect(page.getByRole("button", { name: "Reopen" })).toBeVisible();
  });
});
