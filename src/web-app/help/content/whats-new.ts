import type { WhatsNewEntry } from "../help-types";

/**
 * "What's new", newest first: one entry with every change users will
 * notice (doc/151-help-module.md). The id is YYYY-MM-DD-N-slug, N counting
 * up within a day, so ids sort newest first; the Help button shows a dot
 * until the user has seen the newest. `help` is the entry "Learn more"
 * opens.
 */
export const WHATS_NEW: WhatsNewEntry[] = [
  {
    id: "2026-10-10-1-help",
    date: "2026-10-10",
    title: "Help is built in",
    body: `Press **?** or click **Help** at the top right for help with the
page you are on. Words with a dotted underline and the small **?** icons
explain themselves, and **Cmd+K** (Ctrl+K on Windows) jumps to any page or
help topic.`,
    help: "getting-around",
  },
];
