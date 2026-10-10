import type { HelpTopic } from "../help-types";

/**
 * Task-oriented articles ("how do I ...", "how does ... work";
 * doc/151-help-module.md). `summary` is one or two sentences, at most 35
 * words; `body` is help-markdown.ts's subset. `terms` lists the glossary
 * ids the article leans on; `audience` puts it under "Suggested for you"
 * for those users (help/content/audience.ts).
 */
export const TOPICS: HelpTopic[] = [
  {
    id: "getting-around",
    title: "Getting around the app",
    summary:
      "The menu, the Help panel, the ? key and the Cmd+K box: how to find " +
      "a page or an answer quickly.",
    body: `- The **menu** on the left lists the pages you can use; on a
  phone, open it with the button at the top left.
- **Help** at the top right (or the **?** key anywhere outside a text
  box) opens help for the page you are on. Search it, or open a topic.
- **Cmd+K** (Ctrl+K on Windows) opens a box to jump to any page or help
  topic by typing a few letters.
- Your name at the top right opens your [Profile](/userPreferences) and
  **Logout**.

Words with a dotted underline and small **?** icons explain themselves:
hover, tab to them, or click.`,
    audience: ["everyone"],
  },
  {
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
  },
  {
    id: "managing-groups-and-roles",
    title: "Managing groups and roles",
    summary:
      "Access is given by groups: each group holds roles, and users get " +
      "the roles of every group they are in.",
    body: `On [User Groups](/appUserGroupList), open a group (or **Add User
Group**). **Roles** lists what its members may do; under **Group Members**,
**Add Users** and **Remove Users** change who is in it.

A group still in use cannot be deleted.`,
    terms: ["user-group", "grant"],
    audience: ["everyone"],
  },
  {
    id: "signing-in-with-mfa",
    title: "Signing in with MFA",
    summary:
      "Your password, then a six-digit code from your authenticator app. " +
      "The first time, you scan a QR code to set it up.",
    body: `1. Sign in with your username and password.
2. The first time, scan the QR code with an authenticator app on your
   phone (or type the key shown below it).
3. Enter the six-digit code the app shows. Codes change every 30 seconds;
   use the current one.

A code refused? Check the time on your phone is set automatically. Lost
the phone? Ask an administrator to **Reset MFA** for you.`,
    terms: ["mfa"],
    audience: ["everyone"],
  },
  {
    id: "switching-security-context",
    title: "Switching to another organisation",
    summary:
      "If you belong to more than one security context, change which one " +
      "you are working in from your Profile page.",
    body: `On your [Profile](/userPreferences) page, under **Security**,
pick the context and click **Change Context**. The app reloads its data
for that context, and your roles become the ones you hold there.

With only one context the page says no others are available.`,
    terms: ["security-context", "grant"],
  },
  {
    id: "working-with-examples",
    title: "Working with examples",
    summary:
      "List, sort, add, edit and delete example records: the pattern every " +
      "kind of record in the app follows.",
    body: `- On [Examples](/exampleList), click a column heading to sort;
  the page and sort stay in the address, so a link keeps your place.
- **Add Example** opens an empty form; **Edit** opens one record.
- **Save** refuses to overwrite a change someone else saved since you
  opened the record: reload it and make your change again.
- **Delete** asks first.

How many rows a page shows is your **Rows per page** preference on your
[Profile](/userPreferences).`,
    terms: ["example-record"],
    audience: ["everyone"],
  },
];
