import type { HelpTerm } from "../help-types";

/**
 * The glossary: one entry per word this app uses in a sense of its own
 * (doc/151-help-module.md). `short` is the hover text: one sentence, at
 * most 30 words. `body` is the panel's full entry in help-markdown.ts's
 * subset; a wrapped list item continues on an indented line. Ids are
 * lower-case kebab and never shared with a topic. Pages use them as
 * <Term id="...">, <FieldHelp id="...">, <HelpLink id="..."> or a
 * [label](help:<id>) link.
 */
export const GLOSSARY: HelpTerm[] = [
  {
    id: "security-context",
    term: "Security context",
    short:
      "The organisation whose data you are working in. Everything you see " +
      "and change belongs to it.",
    body: `Every record in the app belongs to exactly one security context.
You see only the current context's data, and your [roles](help:grant) come
from your groups in that context.

Most people have one context. If you belong to more than one, switch on
your [Profile](/userPreferences) page under **Security**: the app reloads
its data for the new context.`,
    aliases: ["context", "tenant", "workspace", "organisation"],
    related: ["grant", "switching-security-context"],
  },
  {
    id: "user-group",
    term: "User group",
    short:
      "A named set of roles. Putting a user in a group gives them every " +
      "role the group holds.",
    body: `Groups are how access is managed: give a group its
[roles](help:grant), then add users to it. A user can do what every role
of every group they are in allows.

Manage groups on [User Groups](/appUserGroupList). A group still in use
cannot be deleted.`,
    aliases: ["group", "groups"],
    related: ["grant", "managing-groups-and-roles"],
  },
  {
    id: "grant",
    term: "Role",
    short:
      "Permission to do one kind of thing, such as reading users or " +
      "editing examples. Roles are given to groups, not to people.",
    body: `Each role allows one kind of action: **Read Example**, **Create
User**, **Block User** and so on. A [user group](help:user-group) holds a
list of roles, and its members can do what those roles allow.

The server checks your roles on every action. A page you may not use is
left out of your menu, and opening it anyway takes you back to the
dashboard with a message.`,
    aliases: ["roles", "grants", "permission", "permissions", "access"],
    related: ["user-group", "managing-groups-and-roles"],
  },
  {
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
  },
  {
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
  },
  {
    id: "mfa",
    term: "MFA",
    short:
      "Multi-factor authentication: after your password, a six-digit code " +
      "from an authenticator app on your phone.",
    body: `On your first sign-in you scan a QR code with an authenticator
app (any app that shows TOTP codes). From then on each sign-in asks for
the six-digit code it shows at that moment.

Lost your phone, or the codes stopped working? An administrator can
**Reset MFA** on your user page; you scan a new code at your next
sign-in.`,
    aliases: [
      "two-factor",
      "2fa",
      "authenticator",
      "totp",
      "code",
      "multi-factor",
    ],
    related: ["signing-in-with-mfa"],
  },
  {
    id: "example-record",
    term: "Example",
    short:
      "The template's sample record: a little of every kind of field, to " +
      "show how a list and an edit page work.",
    body: `An example has text, whole and decimal numbers, lists of
numbers, a JSON value and a UUID the server assigns. It shows the patterns
every record in the app follows: a sortable, paged list, an edit page, and
a save that refuses to overwrite someone else's newer change.`,
    aliases: ["examples", "record"],
    related: ["working-with-examples"],
  },
];
