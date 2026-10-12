import type { HelpTopic } from "../../help-types";

export default {
  id: "getting-around",
  title: "Getting around the app",
  summary:
    "The menu, the Help panel, the ? key and the Cmd+K box: how to find " +
    "a page or an answer quickly.",
  body: `- The **menu** on the left lists the pages you can use; on a
  phone, open it with the button at the top left.
- **Help** at the top right (or the **?** key anywhere outside a text
  box) opens help for the page you are on. Search it, or open a topic.
- **Cmd+K** (Ctrl+K on Windows) opens a box to jump to any page or help
  topic by typing a few letters.
- Your name at the top right opens your [Profile](/userPreferences) and
  **Logout**.

Words with a dotted underline and small **?** icons explain themselves:
hover, tab to them, or click.`,
  audience: ["everyone"],
} satisfies HelpTopic;
