import type { HelpTerm } from "../../help-types";

export default {
  id: "device-token",
  term: "Device token",
  short:
    "The secret a base station checks in with. Shown once when issued; " +
    "issuing a new one stops the old one working.",
  body: `The server keeps only a fingerprint of the token, so it is shown
once, when it is issued: copy it into the base station's config file then.

Lost it? Issue a new one. The old token stops working at once, so a base
station still using it can no longer check in until it has the new one.`,
  aliases: ["token", "device secret"],
  related: ["base-station", "setting-up-a-device"],
} satisfies HelpTerm;
