import { expect, test, type Page } from "@playwright/test";

// The monitoring pages end to end, as the admin: set up a base station
// (device, band, contact, token), a contact's reply through the text
// simulator (the e2e API runs SMS_PROVIDER=log), and the tenant settings'
// first save. Serial: each step builds on the device the first one made.

test.describe.configure({ mode: "serial" });

const DEVICE_NAME = "e2e base station";
const CONTACT_PHONE_TYPED = "704.860.9088";
const CONTACT_PHONE = "+17048609088";
const TOKEN = /^[0-9a-f]{64}$/;

let devicePath = "";

const tokenField = (page: Page) =>
  page.getByRole("dialog").getByRole("textbox", { name: "Device token" });

test.describe("monitoring pages", () => {
  test("create a device, add a band and a contact, issue its token", async ({
    page,
  }) => {
    await page.goto("/deviceEdit?deviceId=new");
    await expect(page.getByRole("heading", { name: "Add Device" }))
      .toBeVisible();
    await page.getByRole("textbox", { name: "Name" }).fill(DEVICE_NAME);
    // The form refuses what the API would (deviceSchema).
    const interval = page.getByRole("textbox", { name: /Check-in interval/ });
    await interval.fill("9");
    await page.getByRole("button", { name: "Save" }).click();
    await expect(page.getByText("At least 10 seconds")).toBeVisible();
    await interval.fill("60");
    await page.getByRole("button", { name: "Save" }).click();

    // A new device stays open: its bands, contacts and token come next.
    await expect(page).toHaveURL(/\/deviceEdit\?deviceId=\d+$/);
    await expect(page.getByRole("heading", { name: "Edit Device" }))
      .toBeVisible();
    devicePath = new URL(page.url()).pathname + new URL(page.url()).search;
    await expect(page.getByRole("textbox", { name: "Last check-in" }))
      .toHaveValue("never");

    // A band, typed in lower case, saved in upper.
    await page.getByRole("button", { name: "Add Band" }).click();
    await expect(page.getByRole("heading", { name: "Add Band" }))
      .toBeVisible();
    await page
      .getByRole("textbox", { name: /Address/ })
      .fill("aa:bb:cc:00:11:22");
    await page.getByRole("textbox", { name: "Label" }).fill("e2e band");
    await page.getByRole("button", { name: "Save" }).click();
    await expect(page).toHaveURL(devicePath);
    await expect(page.getByRole("row", { name: /e2e band/ })).toContainText(
      "AA:BB:CC:00:11:22",
    );

    // A contact, the number typed the US way and shown normalised.
    await page.getByRole("button", { name: "Add Contact" }).click();
    await expect(page.getByRole("heading", { name: "Add Contact" }))
      .toBeVisible();
    await page.getByRole("textbox", { name: "Name" }).fill("e2e contact");
    await page
      .getByRole("textbox", { name: "Phone" })
      .fill(CONTACT_PHONE_TYPED);
    await expect(page.getByText(`Saved as ${CONTACT_PHONE}`)).toBeVisible();
    // Every event starts ticked.
    await expect(page.getByRole("checkbox", { name: "Alarm raised" }))
      .toBeChecked();
    await page.getByRole("button", { name: "Save" }).click();
    await expect(page).toHaveURL(devicePath);
    const contactRow = page.getByRole("row", { name: /e2e contact/ });
    await expect(contactRow).toContainText(CONTACT_PHONE);
    await expect(contactRow).toContainText("Invited (waiting for Y)");

    // The token, shown once, with the warning.
    await page.getByRole("button", { name: "Issue device token" }).click();
    await expect(tokenField(page)).toHaveValue(TOKEN);
    const firstToken = await tokenField(page).inputValue();
    await expect(page.getByRole("dialog")).toContainText(
      "Any token issued before it has stopped working",
    );
    await page.getByRole("button", { name: "Done" }).click();
    await expect(page.getByRole("dialog")).toHaveCount(0);

    // A second token is asked about first and replaces the first.
    await page.getByRole("button", { name: "Issue device token" }).click();
    await page.getByRole("button", { name: "Issue new token" }).click();
    await expect(tokenField(page)).toHaveValue(TOKEN);
    expect(await tokenField(page).inputValue()).not.toBe(firstToken);
    await page.getByRole("button", { name: "Done" }).click();

    // The list: never checked in, one band, no alarms.
    await page.goto("/deviceList");
    const deviceRow = page.getByRole("row", { name: new RegExp(DEVICE_NAME) });
    await expect(deviceRow).toContainText("never");
    await expect(deviceRow.getByRole("cell").nth(3)).toHaveText("1");
    await expect(deviceRow).toContainText("none");
  });

  test("a contact's simulated Y makes them active; their number is then fixed",
    async ({ page }) => {
      await page.goto("/alertList");
      await expect(page.getByRole("heading", { name: "Alerts" }))
        .toBeVisible();
      await page.getByRole("textbox", { name: "From phone" })
        .fill(CONTACT_PHONE_TYPED);
      await page.getByRole("textbox", { name: "Text" }).fill("Y");
      await page.getByRole("button", { name: "Send" }).click();
      await expect(page.getByRole("alert").filter({ hasText: "Reply" }))
        .toContainText(DEVICE_NAME);

      await page.goto(devicePath);
      const contactRow = page.getByRole("row", { name: /e2e contact/ });
      await expect(contactRow).toContainText("Active");
      await contactRow.getByRole("button", { name: "Edit" }).click();
      await expect(page.getByRole("heading", { name: "Edit Contact" }))
        .toBeVisible();
      await page.getByRole("textbox", { name: "Phone" }).fill("7048609089");
      await page.getByRole("button", { name: "Save" }).click();
      // The API's 409, shown as it says it.
      await expect(page.getByText(/their number cannot change/).last())
        .toBeVisible();
      await expect(page).toHaveURL(/\/contactEdit\?contactId=\d+$/);
    });

  test("settings start from the defaults and save", async ({ page }) => {
    await page.goto("/tenantSetting");
    await expect(page.getByText("Not saved yet")).toBeVisible();
    await expect(page.getByRole("textbox", { name: "Time zone" }))
      .toHaveValue("America/New_York");
    const interval = page.getByRole("textbox", { name: /Reminder interval/ });
    await expect(interval).toHaveValue("15");
    await interval.fill("20");
    await page.getByRole("button", { name: "Save" }).click();
    await expect(page.getByText("Settings saved successfully")).toBeVisible();

    await page.reload();
    await expect(page.getByRole("textbox", { name: /Reminder interval/ }))
      .toHaveValue("20");
    await expect(page.getByText("Not saved yet")).toHaveCount(0);
  });
});
