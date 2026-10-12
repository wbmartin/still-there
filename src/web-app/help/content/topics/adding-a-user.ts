import type { HelpTopic } from "../../help-types";

export default {
  id: "adding-a-user",
  title: "Adding a user",
  summary:
    "Create a login and let the person choose their own password with an " +
    "invite link, or set one for them.",
  body: `1. On [User Admin](/appUserList), click **Add User**.
2. Fill in the username, name and email, and pick their
   [groups](help:user-group).
3. Choose **Send an invite link** (best: they choose their own password)
   or **Set a password**.
4. Save. With an invite, copy the link the app shows and send it to them.

Until they use the link the user shows as **Invited**
([account status](help:account-status)). They set up [MFA](help:mfa) at
their first sign-in.`,
  terms: ["invite-link", "account-status", "user-group", "mfa"],
  audience: ["everyone"],
} satisfies HelpTopic;
