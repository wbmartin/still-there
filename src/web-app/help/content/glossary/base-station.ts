import type { HelpTerm } from "../../help-types";

export default {
  id: "base-station",
  term: "Base station",
  short:
    "The small computer in a home that listens for the bands and checks " +
    "in with this server; a device in the app.",
  body: `A base station (a **device** on the Devices page) sits in one
home, listens for the Bluetooth bands people wear, and checks in with this
server every check-in interval. It carries a [device token](help:device-token)
so the server knows which home it is.

If it misses enough check-ins in a row, the server raises a "base station
offline" alarm.`,
  aliases: ["device", "devices", "pi", "hub"],
  related: ["band", "setting-up-a-device"],
} satisfies HelpTerm;
