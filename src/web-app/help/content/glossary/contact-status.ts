import type { HelpTerm } from "../../help-types";

export default {
  id: "contact-status",
  term: "Contact status",
  short:
    "Invited until the contact texts Y back, then active; opted out once " +
    "they text STOP.",
  body: `- **Invited**: added here and sent an invite text; nothing else is
  texted to them until they reply Y.
- **Active**: they replied Y and are texted the events ticked on their
  page.
- **Opted out**: they texted STOP. Carriers block texts to them after
  that, so only their texting START re-enables alerts; nothing here can.

Once a contact has replied, their number cannot change: add a new contact
for a new number.`,
  aliases: ["invited", "opted out", "stop", "start", "contact"],
  related: ["setting-up-a-device"],
} satisfies HelpTerm;
