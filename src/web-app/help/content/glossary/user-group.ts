import type { HelpTerm } from "../../help-types";

export default {
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
} satisfies HelpTerm;
