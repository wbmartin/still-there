import type { HelpTerm } from "../../help-types";

export default {
  id: "grant",
  term: "Role",
  short:
    "Permission to do one kind of thing, such as reading users or " +
    "editing examples. Roles are given to groups, not to people.",
  body: `Each role allows one kind of action: **Read Example**, **Create
User**, **Block User** and so on. A [user group](help:user-group) holds a
list of roles, and its members can do what those roles allow.

The server checks your roles on every action. A page you may not use is
left out of your menu, and opening it anyway takes you back to the
dashboard with a message.`,
  aliases: ["roles", "grants", "permission", "permissions", "access"],
  related: ["user-group", "managing-groups-and-roles"],
} satisfies HelpTerm;
