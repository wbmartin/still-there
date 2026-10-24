import type { HelpTerm } from "../../help-types";

export default {
  id: "band",
  term: "Band",
  short:
    "A Bluetooth band someone wears; the base station reports whether it " +
    "is in range.",
  body: `A band is known by its Bluetooth address (AA:BB:CC:DD:EE:FF) and
given a label, usually whose it is. The base station reports each band as
present or absent, with when it was last seen and its signal strength
(RSSI).

A band unseen for longer than the device's **absent after** time raises a
"band absent" alarm. Seen again, the alarm resolves and the contacts are
told alerting is re-activated.`,
  aliases: ["beacon", "bands", "wristband", "mac address"],
  related: ["base-station", "alarm-life"],
} satisfies HelpTerm;
