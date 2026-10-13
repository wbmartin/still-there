import type { HelpTopic } from "../../help-types";

export default {
  id: "working-with-examples",
  title: "Working with examples",
  summary:
    "List, sort, add, edit and delete example records: the pattern every " +
    "kind of record in the app follows.",
  body: `- On [Examples](/exampleList), click a column heading to sort;
  the page and sort stay in the address, so a link keeps your place.
- **Add Example** opens an empty form; **Edit** opens one record.
- **Save** refuses to overwrite a change someone else saved since you
  opened the record: reload it and make your change again.
- **Delete** asks first.

How many rows a page shows is your **Rows per page** preference on your
[Profile](/userPreferences).`,
  terms: ["example-record"],
  audience: ["everyone"],
} satisfies HelpTopic;
