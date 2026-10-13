import type { HelpTopic } from "../../help-types";

export default {
  id: "switching-security-context",
  title: "Switching to another organisation",
  summary:
    "If you belong to more than one security context, change which one " +
    "you are working in from your Profile page.",
  body: `On your [Profile](/userPreferences) page, under **Security**,
pick the context and click **Change Context**. The app reloads its data
for that context, and your roles become the ones you hold there.

With only one context the page says no others are available.`,
  terms: ["security-context", "grant"],
} satisfies HelpTopic;
