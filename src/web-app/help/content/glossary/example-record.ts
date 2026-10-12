import type { HelpTerm } from "../../help-types";

export default {
  id: "example-record",
  term: "Example",
  short:
    "The template's sample record: a little of every kind of field, to " +
    "show how a list and an edit page work.",
  body: `An example has text, whole and decimal numbers, lists of
numbers, a JSON value and a UUID the server assigns. It shows the patterns
every record in the app follows: a sortable, paged list, an edit page, and
a save that refuses to overwrite someone else's newer change.`,
  aliases: ["examples", "record"],
  related: ["working-with-examples"],
} satisfies HelpTerm;
