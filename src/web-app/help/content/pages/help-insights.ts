import { ROUTES } from "../../../constants/routes";
import type { PageHelp } from "../../help-types";

export default {
  route: ROUTES.HELP_INSIGHTS,
  title: "Help insights",
  purpose:
    "What people search the help for without finding it, which help " +
    "they read, and which tours they finish: where to write help next.",
  steps: [
    "A search with no results is a gap: add a term, an alias or a topic.",
  ],
} satisfies PageHelp;
