import { ROUTES } from "../../../constants/routes";
import type { PageHelp } from "../../help-types";

export default {
  route: ROUTES.ISSUE_REPORT_LIST,
  title: "Issue reports",
  purpose:
    "Every bug, polish and feature report filed in this security " +
    "context, with its status and conversation. File a new one with " +
    "Report at the top of any page.",
  steps: [
    "Open a report to read the replies and add your own, with files.",
    "Withdraw a report that is still open, or reopen a closed one.",
  ],
  terms: ["security-context"],
} satisfies PageHelp;
