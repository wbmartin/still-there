import type { HelpTerm } from "../../help-types";

export default {
  id: "invite-link",
  term: "Invite link",
  short:
    "A one-time link that lets a new user set their own password. It " +
    "expires, and only the newest one works.",
  body: `When you add a user with **Send an invite link**, the app shows
the link once: copy it and send it to them. Opening it lets them choose a
password; after that the link no longer works.

If a link is lost or has expired, open the user and click **New invite
link**; the old one stops working. **Revoke link** cancels it without a
replacement.`,
  aliases: ["invite", "invitation"],
  related: ["account-status", "adding-a-user"],
} satisfies HelpTerm;
