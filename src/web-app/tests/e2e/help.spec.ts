import { expect, test } from "@playwright/test";
import { ROUTES } from "../../constants/routes";

// The header button, with or without its "something new" dot; the page's
// own "Help: ..." field icons are other buttons.
const HELP_BUTTON = { name: /^Help( \(something new\))?$/ };

// The help module end to end (doc/151-help-module.md), with HELP_ENABLED
// on in the e2e API (etc/e2e-api-srv.env): the header button, the panel on
// the page's help, search, the ? key, a ?help= deep link, a page tour and
// the command palette. As the fixture admin; what the panel marks seen is
// that login's own preference, so nothing here depends on its prior state.

test.describe("help", () => {
  test("the Help button opens the page's help and search finds a term", async ({
    page,
  }) => {
    await page.goto(ROUTES.EXAMPLE_LIST);
    await expect(page.getByRole("heading", { name: "Examples" })).toBeVisible();
    await page.getByRole("button", HELP_BUTTON).click();

    const drawer = page.getByRole("dialog");
    await expect(drawer.getByText("On this page")).toBeVisible();
    await expect(
      drawer.getByText("Every example in this security context"),
    ).toBeVisible();
    // Opening the home view marks the newest news seen: the dot goes.
    await expect(page.getByRole("button", { name: "Help", exact: true }))
      .toBeVisible();

    await drawer.getByLabel("Search help").fill("tenant");
    await drawer.getByRole("button", { name: /Security context/ }).click();
    await expect(
      drawer.getByRole("heading", { name: "Security context" }),
    ).toBeVisible();

    await page.keyboard.press("Escape");
    await expect(drawer).toBeHidden();
    await page.getByRole("heading", { name: "Examples" }).click();
    await page.keyboard.press("?");
    await expect(page.getByRole("dialog").getByText("On this page"))
      .toBeVisible();
  });

  test("a ?help= link opens that entry and leaves the address clean", async ({
    page,
  }) => {
    await page.goto(`${ROUTES.DASHBOARD}?help=mfa`);
    const drawer = page.getByRole("dialog");
    await expect(drawer.getByRole("heading", { name: "MFA" })).toBeVisible();
    await expect(page).toHaveURL(new RegExp(`${ROUTES.DASHBOARD}$`));
  });

  test("a page tour walks its steps", async ({ page }) => {
    await page.goto(ROUTES.EXAMPLE_LIST);
    await expect(page.getByRole("heading", { name: "Examples" })).toBeVisible();
    await page.getByRole("button", HELP_BUTTON).click();
    await page.getByRole("button", { name: "Show me around this page" })
      .click();

    const card = page.getByTestId("help-tour-card");
    await expect(card).toContainText("Add an example");
    await expect(card).toContainText("1 of 2");
    await card.getByRole("button", { name: "Next" }).click();
    await expect(card).toContainText("The list");
    await card.getByRole("button", { name: "Done" }).click();
    await expect(card).toBeHidden();
  });

  test("the command palette goes to a page", async ({ page }) => {
    await page.goto(ROUTES.DASHBOARD);
    await expect(page.getByRole("heading", { name: "Dashboard" }))
      .toBeVisible();
    // The palette loads with the rest of the help after sign-in (a lazy
    // chunk): press until it answers.
    const search = page.getByPlaceholder("Go to a page or find help");
    await expect(async () => {
      await page.keyboard.press("ControlOrMeta+k");
      await expect(search).toBeVisible({ timeout: 1000 });
    }).toPass();
    await search.fill("User Groups");
    await page.getByRole("button", { name: /^User Groups/ }).first().click();
    await expect(page).toHaveURL(new RegExp(`${ROUTES.APP_USER_GROUP_LIST}$`));
  });
});
