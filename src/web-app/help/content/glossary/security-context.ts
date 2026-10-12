import type { HelpTerm } from "../../help-types";

export default {
  id: "security-context",
  term: "Security context",
  short:
    "The organisation whose data you are working in. Everything you see " +
    "and change belongs to it.",
  body: `Every record in the app belongs to exactly one security context.
You see only the current context's data, and your [roles](help:grant) come
from your groups in that context.

Most people have one context. If you belong to more than one, switch on
your [Profile](/userPreferences) page under **Security**: the app reloads
its data for the new context.`,
  aliases: ["context", "tenant", "workspace", "organisation"],
  related: ["grant", "switching-security-context"],
} satisfies HelpTerm;
