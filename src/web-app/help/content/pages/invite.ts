import { ROUTES } from "../../../constants/routes";
import type { PageHelp } from "../../help-types";

export default {
  route: ROUTES.INVITE,
  title: "Accept your invite",
  purpose:
    "Choose the password for the login an administrator created for " +
    "you. The link works once.",
  steps: [
    "Link expired or already used? Ask the administrator for a new one.",
  ],
  terms: ["invite-link"],
} satisfies PageHelp;
