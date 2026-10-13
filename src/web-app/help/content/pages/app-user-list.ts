import { ROUTES } from "../../../constants/routes";
import type { PageHelp } from "../../help-types";

export default {
  route: ROUTES.APP_USER_LIST,
  title: "User Admin",
  purpose:
    "Every login in this security context, with its status and groups. " +
    "Add a user, or open one to change, block or reset them.",
  terms: ["account-status", "user-group", "invite-link"],
  topics: ["adding-a-user"],
} satisfies PageHelp;
