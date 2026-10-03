import { expect, test } from "@playwright/test";

// The example entity is the template every derived entity copies, so its
// pages must show the jsonb and uuid columns working end to end.

test.describe("example pages", () => {
  test("create with JSON, see the server uuid, edit the JSON", async ({
    page,
  }) => {
    await page.goto("/example?exampleId=new");
    await page.getByLabel("Text", { exact: true }).fill("e2e jsonb example");
    // Field help is behind an info icon (components/help-tip.tsx); its
    // label also names the icon, so fields are found by role.
    await page.getByRole("img", { name: "About JSON" }).focus();
    await expect(page.getByRole("tooltip")).toHaveText(
      "Any JSON value; leave empty for none",
    );
    await page
      .getByRole("textbox", { name: "JSON" })
      .fill('{"colour": "teal", "count": 3}');
    await page.getByRole("button", { name: "Save" }).click();
    await expect(page).toHaveURL(/\/exampleList/);

    // The new row shows a v4 uuid the server chose.
    const row = page.getByRole("row", { name: /e2e jsonb example/ });
    await expect(row).toBeVisible();
    const uuid = (await row.getByRole("cell").nth(4).textContent()) ?? "";
    expect(uuid).toMatch(
      /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/,
    );

    // The edit page round-trips the JSON and shows the uuid read-only.
    await row.getByRole("button", { name: "Edit" }).click();
    await expect(page.getByRole("textbox", { name: "UUID" })).toHaveValue(
      uuid,
    );
    const jsonField = page.getByRole("textbox", { name: "JSON" });
    expect(JSON.parse(await jsonField.inputValue())).toEqual({
      colour: "teal",
      count: 3,
    });

    // Invalid JSON is refused before it reaches the server.
    await jsonField.fill("{not json");
    await page.getByRole("button", { name: "Save" }).click();
    await expect(page.getByText("Must be valid JSON")).toBeVisible();

    await jsonField.fill("[1, 2, 3]");
    await page.getByRole("button", { name: "Save" }).click();
    await expect(page).toHaveURL(/\/exampleList/);
    await page
      .getByRole("row", { name: /e2e jsonb example/ })
      .getByRole("button", { name: "Edit" })
      .click();
    await expect(page.getByRole("textbox", { name: "UUID" })).toHaveValue(
      uuid,
    );
    const editedJson = await page
      .getByRole("textbox", { name: "JSON" })
      .inputValue();
    expect(JSON.parse(editedJson)).toEqual([
      1, 2, 3,
    ]);

    // Leave the fixtures as they were: the smoke spec counts them.
    await page.getByRole("button", { name: "Delete" }).click();
    await page
      .getByRole("dialog")
      .getByRole("button", { name: "Delete" })
      .click();
    await expect(page).toHaveURL(/\/exampleList/);
    await expect(
      page.getByRole("row", { name: /e2e jsonb example/ }),
    ).toHaveCount(0);
  });

  // A form is never rendered for a record that was not loaded (B076).
  test("an id that does not exist shows an alert, not a form", async ({
    page,
  }) => {
    await page.goto("/example?exampleId=999999999");
    await expect(
      page.getByText("This example could not be loaded."),
    ).toBeVisible();
    await expect(page.getByLabel("Text", { exact: true })).toHaveCount(0);
    await expect(page.getByRole("button", { name: "Save" })).toHaveCount(0);
  });

  // The router reuses the edit page when only ?exampleId= changes; the form
  // must reload for the new id (B076). history.go(-2) from one edit page to
  // another is one such change: a single popstate, the page not remounted.
  test("a new ?exampleId= on the same page reloads the form", async ({
    page,
  }) => {
    const textField = page.getByLabel("Text", { exact: true });
    const openSeed = async (exampleText: string) => {
      await page
        .getByRole("row", { name: new RegExp(`${exampleText}\\b`) })
        .getByRole("button", { name: "Edit" })
        .click();
      await expect(textField).toHaveValue(exampleText);
    };

    await page.goto("/exampleList");
    await openSeed("Sample example 1");
    const firstUrl = page.url();
    await page.getByRole("button", { name: "Cancel" }).click();
    await expect(page).toHaveURL(/\/exampleList/);
    await openSeed("Sample example 2");

    await page.evaluate(() => window.history.go(-2));
    await expect(page).toHaveURL(firstUrl);
    await expect(textField).toHaveValue("Sample example 1");
  });

  // Every tab shares the cookie but keeps its own CSRF token; another tab
  // loading must not leave this one's Save refused (B004).
  test("a save still works after another tab loads", async ({
    page,
    context,
  }) => {
    await page.goto("/example?exampleId=new");
    await page.getByLabel("Text", { exact: true }).fill("e2e two tabs");

    const otherTab = await context.newPage();
    await otherTab.goto("/exampleList");
    await expect(otherTab.getByRole("table")).toBeVisible();
    await otherTab.close();

    await page.getByRole("button", { name: "Save" }).click();
    await expect(page).toHaveURL(/\/exampleList/);
    await page
      .getByRole("row", { name: /e2e two tabs/ })
      .getByRole("button", { name: "Delete" })
      .click();
    await page
      .getByRole("dialog")
      .getByRole("button", { name: "Delete" })
      .click();
    await expect(
      page.getByRole("row", { name: /e2e two tabs/ }),
    ).toHaveCount(0);
  });
});
