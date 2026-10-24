import { ROUTES } from "../../../constants/routes";
import type { PageHelp } from "../../help-types";

export default {
  route: ROUTES.DEVICE_LIST,
  title: "Devices",
  purpose:
    "Every base station in this account: when it last checked in, its " +
    "agent version, how many bands it watches and its open alarms.",
  steps: [
    "A last check-in of never means it has not reported with its token yet.",
    "Click an open-alarm badge to see that device's alarms.",
    "Use Add Device to set up a new base station.",
  ],
  terms: ["base-station", "band"],
  topics: ["setting-up-a-device"],
} satisfies PageHelp;
