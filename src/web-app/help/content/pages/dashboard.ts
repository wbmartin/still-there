import { ROUTES } from "../../../constants/routes";
import type { PageHelp } from "../../help-types";

export default {
  route: ROUTES.DASHBOARD,
  title: "Dashboard",
  purpose:
    "Where you land after signing in, with a count of the examples in " +
    "this security context.",
  terms: ["security-context"],
  topics: ["getting-around"],
} satisfies PageHelp;
