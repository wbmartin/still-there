import { ROUTES } from "../../../constants/routes";
import type { PageHelp } from "../../help-types";

export default {
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
} satisfies PageHelp;
