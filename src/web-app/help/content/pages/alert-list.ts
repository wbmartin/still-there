import { ROUTES } from "../../../constants/routes";
import type { PageHelp } from "../../help-types";

export default {
  route: ROUTES.ALERT_LIST,
  title: "Alerts",
  purpose:
    "Open alarms first, then those resolved in the last week. The page " +
    "refreshes itself every 30 seconds.",
  steps: [
    "Acknowledge tells every contact you are on it; reminders drop to " +
      "one a day.",
    "Turn off tells every contact and stops the reminders.",
    "If someone changed the alarm first, the page says so and reloads it.",
  ],
  terms: ["band", "base-station"],
  topics: ["alarm-life"],
} satisfies PageHelp;
