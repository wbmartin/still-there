import { ROUTES } from "../../../constants/routes";
import type { PageHelp } from "../../help-types";

export default {
  route: ROUTES.DEVICE_EDIT,
  title: "Edit a device",
  purpose:
    "One base station: its name and alarm settings, its token, and the " +
    "bands and contacts that belong to it.",
  steps: [
    "Changing a setting raises the config version; the base station " +
      "picks it up at its next check-in.",
    "Issue device token shows a new token once; the old one stops working.",
    "Add, edit or delete its bands and contacts below the form.",
  ],
  terms: ["base-station", "device-token", "band", "contact-status"],
  topics: ["setting-up-a-device"],
} satisfies PageHelp;
