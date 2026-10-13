import { ROUTES } from "../../../constants/routes";
import type { PageHelp } from "../../help-types";

export default {
  route: ROUTES.EXAMPLE_LIST,
  title: "Examples",
  purpose:
    "Every example in this security context, a page at a time. Sort by " +
    "a column, add one, or open one to edit it.",
  steps: [
    "Click a column heading to sort; click it again to reverse.",
    "Use Add Example to create one.",
  ],
  terms: ["example-record"],
  topics: ["working-with-examples"],
} satisfies PageHelp;
