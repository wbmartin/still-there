import { ROUTES } from "../../../constants/routes";
import type { PageHelp } from "../../help-types";

export default {
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
} satisfies PageHelp;
