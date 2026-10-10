import { ROUTES } from "../../constants/routes";
import type { PageHelp } from "../help-types";

/**
 * Help for each routed page, keyed by its ROUTES value: what the Help
 * panel shows under "On this page" (doc/151-help-module.md).
 * tests/help-content.test.ts fails while a route in constants/routes.ts
 * has no entry here and is not in PAGES_WITHOUT_HELP, so a new page ships
 * with its help in the same change.
 */
export const PAGE_HELP: PageHelp[] = [
  {
    route: ROUTES.LOGIN,
    title: "Sign in",
    purpose:
      "Sign in with your username and password, then the code from your " +
      "authenticator app.",
    steps: [
      "Too many wrong tries lock the login for a few minutes; wait and " +
        "try again.",
      "Forgot your password? Ask an administrator for a new invite link " +
        "or a new password.",
    ],
    terms: ["mfa"],
    topics: ["signing-in-with-mfa"],
  },
  {
    route: ROUTES.MFA_SETUP,
    title: "Set up MFA",
    purpose:
      "Link an authenticator app to your login: scan the QR code, then " +
      "enter the six-digit code it shows.",
    steps: [
      "Can't scan? Copy the key under the code into the app instead.",
    ],
    terms: ["mfa"],
    topics: ["signing-in-with-mfa"],
  },
  {
    route: ROUTES.MFA_VERIFY,
    title: "Enter your code",
    purpose:
      "The second step of signing in: the six-digit code your " +
      "authenticator app shows now.",
    terms: ["mfa"],
    topics: ["signing-in-with-mfa"],
  },
  {
    route: ROUTES.INVITE,
    title: "Accept your invite",
    purpose:
      "Choose the password for the login an administrator created for " +
      "you. The link works once.",
    steps: [
      "Link expired or already used? Ask the administrator for a new one.",
    ],
    terms: ["invite-link"],
  },
  {
    route: ROUTES.DASHBOARD,
    title: "Dashboard",
    purpose:
      "Where you land after signing in, with a count of the examples in " +
      "this security context.",
    terms: ["security-context"],
    topics: ["getting-around"],
  },
  {
    route: ROUTES.EXAMPLE_LIST,
    title: "Examples",
    purpose:
      "Every example in this security context, a page at a time. Sort by " +
      "a column, add one, or open one to edit it.",
    steps: [
      "Click a column heading to sort; click it again to reverse.",
      "Use Add Example to create one.",
    ],
    terms: ["example-record"],
    topics: ["working-with-examples"],
  },
  {
    route: ROUTES.EXAMPLE_EDIT,
    title: "Edit an example",
    purpose:
      "Create or change one example. The UUID is assigned by the server " +
      "when the example is first saved.",
    steps: [
      "Lists of numbers are typed comma-separated.",
      "Save refuses a change someone else saved since you opened it: " +
        "reload and try again.",
    ],
    terms: ["example-record"],
    topics: ["working-with-examples"],
  },
  {
    route: ROUTES.APP_USER_LIST,
    title: "User Admin",
    purpose:
      "Every login in this security context, with its status and groups. " +
      "Add a user, or open one to change, block or reset them.",
    terms: ["account-status", "user-group", "invite-link"],
    topics: ["adding-a-user"],
  },
  {
    route: ROUTES.APP_USER_EDIT,
    title: "Edit a user",
    purpose:
      "One login: name, email and groups, and the actions on it: change " +
      "password, reset MFA, block, and invite links.",
    steps: [
      "Groups decide what the user can do.",
      "Block stops them signing in at once; Unblock undoes it.",
      "Reset MFA makes them scan a new code at their next sign-in.",
    ],
    terms: ["account-status", "invite-link", "user-group", "mfa"],
    topics: ["adding-a-user"],
  },
  {
    route: ROUTES.APP_USER_GROUP_LIST,
    title: "User Groups",
    purpose:
      "The groups of this security context and the roles each one gives " +
      "its members.",
    terms: ["user-group", "grant"],
    topics: ["managing-groups-and-roles"],
  },
  {
    route: ROUTES.APP_USER_GROUP_EDIT,
    title: "Edit a user group",
    purpose:
      "A group's name, its roles, and who is in it.",
    steps: [
      "Roles: what every member may do.",
      "Add Users and Remove Users change the members.",
    ],
    terms: ["user-group", "grant"],
    topics: ["managing-groups-and-roles"],
  },
  {
    route: ROUTES.USER_PREFERENCES,
    title: "Profile",
    purpose:
      "Your name, email and preferences, your password, and which " +
      "security context you are working in.",
    steps: [
      "Colour scheme and rows per page apply wherever you sign in.",
      "Change Password needs your current password.",
    ],
    terms: ["security-context"],
    topics: ["switching-security-context"],
  },
  {
    route: ROUTES.ISSUE_REPORT_LIST,
    title: "Issue reports",
    purpose:
      "Every bug, polish and feature report filed in this security " +
      "context, with its status and conversation. File a new one with " +
      "Report at the top of any page.",
    steps: [
      "Open a report to read the replies and add your own, with files.",
      "Withdraw a report that is still open, or reopen a closed one.",
    ],
    terms: ["security-context"],
  },
  {
    route: ROUTES.HELP_INSIGHTS,
    title: "Help insights",
    purpose:
      "What people search the help for without finding it, which help " +
      "they read, and which tours they finish: where to write help next.",
    steps: [
      "A search with no results is a gap: add a term, an alias or a topic.",
    ],
  },
];

/**
 * Routes that deliberately have no page help. Empty in the template;
 * name the reason beside any route added here.
 */
export const PAGES_WITHOUT_HELP: string[] = [];
