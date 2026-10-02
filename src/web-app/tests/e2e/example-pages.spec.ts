import { expect, test } from "@playwright/test";

// The example entity is the template every derived entity copies, so its
// pages must show the jsonb and uuid columns working end to end.

test.describe("example pages", () => {
  test("create with JSON, see the server uuid, edit the JSON", async ({
    page,
  }) => {
    await page.goto("/example?exampleId=new");
    await page.getByLabel("Text", { exact: true }).fill("e2e jsonb example");
    await page.getByLabel("JSON").fill('{"colour": "teal", "count": 3}');
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
    await expect(page.getByLabel("UUID")).toHaveValue(uuid);
    const jsonField = page.getByLabel("JSON");
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
    await expect(page.getByLabel("UUID")).toHaveValue(uuid);
    expect(JSON.parse(await page.getByLabel("JSON").inputValue())).toEqual([
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
});
