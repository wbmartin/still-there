import { ROUTES } from "../../../constants/routes";
import type { PageHelp } from "../../help-types";

export default {
  route: ROUTES.EXAMPLE_EDIT,
  title: "Edit an example",
  purpose:
    "Create or change one example. The UUID is assigned by the server " +
    "when the example is first saved.",
  steps: [
    "Lists of numbers are typed comma-separated.",
    "Save refuses a change someone else saved since you opened it: " +
      "reload and try again.",
  ],
  terms: ["example-record"],
  topics: ["working-with-examples"],
} satisfies PageHelp;
