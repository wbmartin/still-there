import { ROUTES } from "../../../constants/routes";
import type { PageHelp } from "../../help-types";

export default {
  route: ROUTES.APP_USER_GROUP_LIST,
  title: "User Groups",
  purpose:
    "The groups of this security context and the roles each one gives " +
    "its members.",
  terms: ["user-group", "grant"],
  topics: ["managing-groups-and-roles"],
} satisfies PageHelp;
