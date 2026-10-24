import { ROUTES } from "../../../constants/routes";
import type { PageHelp } from "../../help-types";

export default {
  route: ROUTES.TENANT_SETTING,
  title: "Settings",
  purpose:
    "When and how often alarm reminders are texted, for every device in " +
    "this account.",
  steps: [
    "The reminder interval repeats a raised alarm until someone " +
      "acknowledges it.",
    "The daily reminder time is when an acknowledged alarm is texted " +
      "again, in the time zone chosen.",
  ],
  topics: ["alarm-life"],
} satisfies PageHelp;
