import type { HelpTopic } from "../../help-types";

export default {
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
} satisfies HelpTopic;
