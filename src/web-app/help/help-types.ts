// The help module's content model (doc/151-help-module.md). Content is
// typed data in help/content/, versioned with the code, and every help
// surface (inline terms, the Help panel, search, tours, the command
// palette) reads it through help-registry.ts.
//
// Every `body` is the safe markdown subset help-markdown.ts parses:
// blank-line paragraphs, "- " bullets, "1. " numbered lists, **bold**,
// [label](help:<id>) links to another entry and [label](/route) links to
// a page of this app. No raw HTML: anything else is shown as text.
import type { HelpAudience } from "./content/audience";

/** A glossary term. `short` is the hover text: one plain sentence. */
export interface HelpTerm {
  id: string;
  term: string;
  short: string;
  body: string;
  // Other words people search for it by.
  aliases?: string[];
  // Term or topic ids shown as "Related".
  related?: string[];
  audience?: HelpAudience[];
}

/** A task-oriented article ("how do I...", "how does ... work"). */
export interface HelpTopic {
  id: string;
  title: string;
  summary: string;
  body: string;
  // Term ids shown as "Terms used here".
  terms?: string[];
  related?: string[];
  audience?: HelpAudience[];
}

/** Help for one routed page, keyed by its ROUTES value. */
export interface PageHelp {
  route: string;
  title: string;
  purpose: string;
  // What to do here, one short sentence each.
  steps?: string[];
  terms?: string[];
  topics?: string[];
}

/** One step of a page tour: points at the element with data-tour=target. */
export interface TourStep {
  target: string;
  title: string;
  body: string;
}

/**
 * A short page tour. Replayed from the Help panel ("Show me around this
 * page"); with autoStart it also starts once on a user's first visit.
 */
export interface Tour {
  id: string;
  route: string;
  title: string;
  autoStart?: boolean;
  steps: TourStep[];
}

/** One "What's new" entry; `help` is the entry "Learn more" opens. */
export interface WhatsNewEntry {
  // YYYY-MM-DD-N-slug, N counting up within a day: sorts newest first.
  id: string;
  date: string;
  title: string;
  body: string;
  help?: string;
}
