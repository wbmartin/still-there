import { expect, test } from "@playwright/test";
import { expectDashboard, loginThroughForm, session } from "./fixtures/auth";
import { api } from "./fixtures/api-client";
import { totpCode } from "./fixtures/totp";

// The TOTP second factor through the real screens (PS0020): password →
// /mfa-setup shows the secret → a wrong code is refused → the right code
// signs in → a second sign-in verifies against the same secret. The
// fixture user `mfa` is not exempt and starts un-enrolled; the admin API
// session resets it at the end so the run is repeatable.

test.describe("mfa", () => {
  test.use({ storageState: { cookies: [], origins: [] } });

  test("enrol, verify, sign in again", async ({ page }) => {
    await loginThroughForm(page, "mfa", "basic123!");
    await expect(page).toHaveURL(/\/mfa-setup/);
    await expect(page.getByAltText(/QR Code/)).toBeVisible();
    const secretText = await page.getByText(/^Secret Key: /).textContent();
    const secret = (secretText ?? "").replace("Secret Key: ", "").trim();
    expect(secret).toMatch(/^[A-Z2-7]{16,}$/);

    await page.getByLabel("Enter TOTP Code").fill("000000");
    await page.getByRole("button", { name: "Verify Code" }).click();
    await expect(page).toHaveURL(/\/mfa-setup/);

    await page.getByLabel("Enter TOTP Code").fill(totpCode(secret));
    await page.getByRole("button", { name: "Verify Code" }).click();
    await expectDashboard(page);

    // Sign out; the second sign-in is verify only, no QR.
    await page.getByRole("button", { name: "Logout" }).click();
    await expect(page).toHaveURL(/\/login/);
    await loginThroughForm(page, "mfa", "basic123!");
    await expect(page).toHaveURL(/\/mfa-setup/);
    await expect(page.getByAltText(/QR Code/)).toHaveCount(0);
    await page.getByLabel("Enter TOTP Code").fill(totpCode(secret));
    await page.getByRole("button", { name: "Verify Code" }).click();
    await expectDashboard(page);

    // Put the fixture back: un-enrol through the admin session.
    const admin = session();
    const userList = await api.get<{ userName: string; appUserId: number }[]>(
      admin,
      "/userList",
    );
    const mfaUser = userList.find((user) => user.userName === "mfa");
    expect(mfaUser).toBeDefined();
    await api.post(admin, "/admin/resetUserMFA", { userId: mfaUser?.appUserId });
  });
});
