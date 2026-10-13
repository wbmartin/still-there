import { ROUTES } from "../../constants/routes";
import type { Tour } from "../help-types";

/**
 * Short page tours (doc/151-help-module.md). Each step points at the
 * element whose data-tour attribute is its target; a step whose element
 * is not on the page is passed over. A tour is replayed from the Help
 * panel ("Show me around this page"); with autoStart it also starts once
 * on a user's first visit. tests/help-content.test.ts fails while a
 * target is not written as data-tour="<target>" in a page or component.
 */
export const TOURS: Tour[] = [
  {
    id: "example-list",
    route: ROUTES.EXAMPLE_LIST,
    title: "The examples list",
    steps: [
      {
        target: "example-add",
        title: "Add an example",
        body: "Opens an empty form. The server assigns the ID and UUID.",
      },
      {
        target: "example-table",
        title: "The list",
        body:
          "Click a column heading to sort, Edit to change a row. The page " +
          "and sort stay in the address, so a link keeps your place.",
      },
    ],
  },
];
