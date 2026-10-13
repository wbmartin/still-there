import { ROUTES } from "../../../constants/routes";
import type { PageHelp } from "../../help-types";

export default {
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
} satisfies PageHelp;
