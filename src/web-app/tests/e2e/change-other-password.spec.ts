import { expect, test } from "@playwright/test";
import { api, signIn } from "./fixtures/api-client";
import { session } from "./fixtures/auth";

// An admin changes another user's password from the edit page. The store
// once posted a key the handler does not read, so every attempt 400'd
// (B013); signing in with the new password proves it reached the server.

interface CreatedUser {
  appUserId: number;
  userName: string;
  updateTs: number;
}

test("an admin changes another user's password", async ({ page }) => {
  const suffix = Date.now().toString(36);
  const newPassword = `e2e-n3w-${suffix}!`;
  const user = await api.post<CreatedUser>(session(), "/user", {
    userName: `e2epw${suffix}`,
    firstName: "Password",
    lastName: "Change",
    email: `e2epw${suffix}@example.com`,
    pwHash: `e2e-old-${suffix}!`,
    mfaExempt: true,
    appUserGroupMembershipList: [],
  });

  try {
    await page.goto(`/appUserEdit?appUserId=${user.appUserId}`);
    await expect(page.getByLabel("Username")).toHaveValue(user.userName);

    await page.getByRole("button", { name: "Change Password" }).click();
    const dialog = page.getByRole("dialog");
    // The visible label carries the required "*"; the accessible name does
    // not.
    await dialog
      .getByRole("textbox", { name: "New Password", exact: true })
      .fill(newPassword);
    await dialog
      .getByRole("textbox", { name: "Confirm New Password" })
      .fill(newPassword);
    await dialog.getByRole("button", { name: "Change Password" }).click();

    await expect(page.getByText("Password changed successfully")).toBeVisible();
    await signIn(user.userName, newPassword);
  } finally {
    const [current] = await api.get<CreatedUser[]>(
      session(),
      `/userList?appUserIdList=${user.appUserId}`,
    );
    await api.delete(
      session(),
      `/user?appUserId=${user.appUserId}&updateTs=${current.updateTs}`,
    );
  }
});
