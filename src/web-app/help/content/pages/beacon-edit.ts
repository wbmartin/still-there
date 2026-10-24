import { ROUTES } from "../../../constants/routes";
import type { PageHelp } from "../../help-types";

export default {
  route: ROUTES.BEACON_EDIT,
  title: "Edit a band",
  purpose:
    "One band: its Bluetooth address and label, and what the base station " +
    "last reported about it.",
  steps: [
    "Type the address as AA:BB:CC:DD:EE:FF; it is saved in capitals.",
    "Present, last seen and RSSI come from the base station and are " +
      "read-only.",
  ],
  terms: ["band", "base-station"],
  topics: ["setting-up-a-device"],
} satisfies PageHelp;
