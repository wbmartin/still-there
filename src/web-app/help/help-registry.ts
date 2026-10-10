// The one way into the help content (doc/151-help-module.md): lookups by
// id and route, the audience for a grant list, and the panel's search.
// Content lives in help/content/; nothing here is app-specific.
import {
  DEFAULT_HELP_AUDIENCE,
  HELP_AUDIENCE_BY_GRANT,
  type HelpAudience,
} from "./content/audience";
import { GLOSSARY } from "./content/glossary";
import { PAGE_HELP, PAGES_WITHOUT_HELP } from "./content/pages";
import { TOPICS } from "./content/topics";
import { TOURS } from "./content/tours";
import { WHATS_NEW } from "./content/whats-new";
import type { HelpTerm, HelpTopic, PageHelp, Tour } from "./help-types";

export { GLOSSARY, PAGE_HELP, PAGES_WITHOUT_HELP, TOPICS, TOURS, WHATS_NEW };

const termById = new Map(GLOSSARY.map((term) => [term.id, term]));
const topicById = new Map(TOPICS.map((topic) => [topic.id, topic]));
const pageByRoute = new Map(PAGE_HELP.map((page) => [page.route, page]));

export const getTerm = (id: string): HelpTerm | undefined =>
  termById.get(id);

export const getTopic = (id: string): HelpTopic | undefined =>
  topicById.get(id);

export const getPageHelp = (route: string): PageHelp | undefined =>
  pageByRoute.get(route);

export const tourForRoute = (route: string): Tour | undefined =>
  TOURS.find((tour) => tour.route === route);

export const getTour = (id: string): Tour | undefined =>
  TOURS.find((tour) => tour.id === id);

/** The newest "What's new" id; undefined while there is none. */
export const latestWhatsNewId = (): string | undefined => WHATS_NEW[0]?.id;

/** A help entry id names a term or a topic; the two never share an id. */
export type HelpEntry =
  | { kind: "term"; item: HelpTerm }
  | { kind: "topic"; item: HelpTopic };

export function getEntry(id: string): HelpEntry | undefined {
  const term = termById.get(id);
  if (term) return { kind: "term", item: term };
  const topic = topicById.get(id);
  if (topic) return { kind: "topic", item: topic };
  return undefined;
}

/** The entry's heading: a term's word, a topic's title. */
export const entryTitle = (entry: HelpEntry): string =>
  entry.kind === "term" ? entry.item.term : entry.item.title;

/** The entry's one-liner: a term's short text, a topic's summary. */
export const entrySummary = (entry: HelpEntry): string =>
  entry.kind === "term" ? entry.item.short : entry.item.summary;

/** The audience of the first HELP_AUDIENCE_BY_GRANT grant held. */
export function audienceFromGrants(grantList: number[]): HelpAudience {
  const match = HELP_AUDIENCE_BY_GRANT.find((row) =>
    grantList.includes(row.grant),
  );
  return match ? match.audience : DEFAULT_HELP_AUDIENCE;
}

export interface HelpSearchResult {
  id: string;
  kind: "term" | "topic";
  title: string;
  summary: string;
  score: number;
}

// Lower case, accents folded, anything but letters and digits a space.
const normalize = (text: string) =>
  text
    .toLowerCase()
    .normalize("NFKD")
    .replace(/\p{M}+/gu, "")
    .replace(/[^\p{L}\p{N}]+/gu, " ")
    .trim();

/**
 * A small ranked search over terms and topics. Every query word must
 * appear in the entry; a word in the title outranks one in an alias, then
 * the summary, then the body. An exact title or alias match comes first,
 * and a definition before a topic on a tie.
 */
export function searchHelp(query: string, limit = 12): HelpSearchResult[] {
  const normalQuery = normalize(query);
  if (!normalQuery) return [];
  const wordList = normalQuery.split(" ");
  const resultList: HelpSearchResult[] = [];
  const consider = (
    id: string,
    kind: "term" | "topic",
    title: string,
    summary: string,
    body: string,
    aliasList: string[],
  ) => {
    const normalTitle = normalize(title);
    const normalAliasList = aliasList.map(normalize);
    const normalAliases = normalAliasList.join(" | ");
    const normalSummary = normalize(summary);
    const normalBody = normalize(body);
    let score = 0;
    for (const word of wordList) {
      if (normalTitle.includes(word)) score += 10;
      else if (normalAliases.includes(word)) score += 8;
      else if (normalSummary.includes(word)) score += 4;
      else if (normalBody.includes(word)) score += 1;
      else return;
    }
    if (
      normalTitle === normalQuery ||
      normalAliasList.includes(normalQuery)
    ) {
      score += 20;
    }
    if (kind === "term") score += 1;
    resultList.push({ id, kind, title, summary, score });
  };
  for (const term of GLOSSARY) {
    consider(term.id, "term", term.term, term.short, term.body,
      term.aliases ?? []);
  }
  for (const topic of TOPICS) {
    consider(topic.id, "topic", topic.title, topic.summary, topic.body, []);
  }
  return resultList
    .sort((a, b) => b.score - a.score || a.title.localeCompare(b.title))
    .slice(0, limit);
}
