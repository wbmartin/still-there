import type { HelpTerm } from "../../help-types";

export default {
  id: "account-status",
  term: "Account status",
  short:
    "Whether a login can sign in: Active, Invited (its invite link not " +
    "used yet) or Blocked.",
  body: `- **Active**: the user signs in with their password and, unless
  exempt, their [MFA](help:mfa) code.
- **Invited**: created with an [invite link](help:invite-link) that has
  not been used yet. The login cannot sign in until its holder sets a
  password through the link.
- **Blocked**: cannot sign in, and any open session ends at once.
  **Unblock** on the user's page undoes it.`,
  aliases: ["status", "blocked", "invited", "active"],
  related: ["invite-link", "adding-a-user"],
} satisfies HelpTerm;
