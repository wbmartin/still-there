import type { HelpTopic } from "../../help-types";

export default {
  id: "setting-up-a-device",
  title: "Setting up a base station",
  summary:
    "Add the device, its bands and the people to text, then issue its " +
    "token and put it in the base station's config file.",
  body: `1. On [Devices](/deviceList), **Add Device**: a name and its alarm
   settings (the defaults suit most homes).
2. On the device's page, **Add Band** for each band, by its Bluetooth
   address, with a label such as whose it is.
3. **Add Contact** for each person to text. They get an invite and are
   texted alarms only after replying Y.
4. **Issue device token**, copy it into the base station's config file,
   and start it. Its last check-in then shows on the Devices page.

Reminder timing for the whole account is on [Settings](/tenantSetting).`,
  terms: ["base-station", "band", "device-token", "contact-status"],
  related: ["alarm-life"],
  audience: ["everyone"],
} satisfies HelpTopic;
