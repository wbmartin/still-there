import { readdirSync, readFileSync } from "node:fs";
import { join, resolve } from "node:path";
import { describe, expect, it } from "vitest";
import { ROUTES } from "../constants/routes";
import { HELP_AUDIENCE_LIST } from "../help/content/audience";
import { helpLinkIdList, routeLinkList } from "../help/help-markdown";
import {
  GLOSSARY,
  PAGE_HELP,
  PAGES_WITHOUT_HELP,
  TOPICS,
  TOURS,
  WHATS_NEW,
  getEntry,
} from "../help/help-registry";

// The help content gates (doc/151-help-module.md): a missing or broken
// piece of help fails the run the way a type error does, so a page ships
// with its help in the same change. They run whether HELP_ENABLED is on
// or off: help stays authored while it is switched off.

const webAppRoot = resolve(import.meta.dirname, "..");

// Every .tsx under pages/, components/ and help/, and app.tsx: where help
// ids and tour anchors are written.
function uiSourceMap(): Map<string, string> {
  const sourceMap = new Map<string, string>();
  for (const directory of ["pages", "components", "help"]) {
    const entryList = readdirSync(join(webAppRoot, directory), {
      recursive: true,
      encoding: "utf8",
    });
    for (const entry of entryList) {
      if (!entry.endsWith(".tsx")) continue;
      const path = join(directory, entry);
      sourceMap.set(path, readFileSync(join(webAppRoot, path), "utf8"));
    }
  }
  sourceMap.set("app.tsx", readFileSync(join(webAppRoot, "app.tsx"), "utf8"));
  return sourceMap;
}

const sourceMap = uiSourceMap();
const routeList: string[] = Object.values(ROUTES);
const termIdSet = new Set(GLOSSARY.map((term) => term.id));
const topicIdSet = new Set(TOPICS.map((topic) => topic.id));
const HELP_ID = /^[a-z0-9][a-z0-9-]{0,63}$/;

// Each component that takes a help id, and openHelp("..."), written with
// a literal id. A computed id is not checked here: keep them literal.
const USED_ID_PATTERN_LIST = [
  /<(?:Term|FieldHelp|HelpLink)\s[^>]*?\bid="([^"]+)"/g,
  /<Nudge\s[^>]*?\bhelp="([^"]+)"/g,
  /openHelp\(\s*"([^"]+)"\s*\)/g,
];
const TOUR_ANCHOR = /data-tour="([^"]+)"/g;

// Each body with where it came from, for the messages.
const bodyList: { from: string; body: string }[] = [
  ...GLOSSARY.map((term) => ({ from: `term ${term.id}`, body: term.body })),
  ...TOPICS.map((topic) => ({ from: `topic ${topic.id}`, body: topic.body })),
  ...WHATS_NEW.map((news) => ({ from: `news ${news.id}`, body: news.body })),
];

const wordCount = (text: string) => text.trim().split(/\s+/).length;

describe("help content", () => {
  it("has content to check (floor)", () => {
    expect(GLOSSARY.length).toBeGreaterThan(0);
    expect(TOPICS.length).toBeGreaterThan(0);
    expect(PAGE_HELP.length).toBeGreaterThan(0);
    expect(sourceMap.size, "UI source scan found no files").toBeGreaterThan(
      10,
    );
  });

  it("uses well-formed ids, unique across terms and topics", () => {
    const idList = [...termIdSet, ...topicIdSet];
    expect(idList.length).toBe(GLOSSARY.length + TOPICS.length);
    expect(new Set(idList).size, "a term and a topic share an id").toBe(
      idList.length,
    );
    for (const id of [...idList, ...TOURS.map((tour) => tour.id)]) {
      expect(id, `id "${id}"`).toMatch(HELP_ID);
    }
    expect(new Set(TOURS.map((tour) => tour.id)).size).toBe(TOURS.length);
  });

  it("keeps hover text and summaries short, and bodies non-empty", () => {
    for (const term of GLOSSARY) {
      expect(wordCount(term.short), `${term.id} short`).toBeLessThanOrEqual(
        30,
      );
      expect(term.body.trim(), `${term.id} body`).not.toBe("");
    }
    for (const topic of TOPICS) {
      expect(
        wordCount(topic.summary),
        `${topic.id} summary`,
      ).toBeLessThanOrEqual(35);
      expect(topic.body.trim(), `${topic.id} body`).not.toBe("");
    }
  });

  it("resolves every related, terms and topics reference", () => {
    for (const term of GLOSSARY) {
      for (const id of term.related ?? []) {
        expect(getEntry(id), `term ${term.id} related ${id}`).toBeDefined();
      }
    }
    for (const topic of TOPICS) {
      for (const id of topic.related ?? []) {
        expect(getEntry(id), `topic ${topic.id} related ${id}`).toBeDefined();
      }
      for (const id of topic.terms ?? []) {
        expect(termIdSet.has(id), `topic ${topic.id} term ${id}`).toBe(true);
      }
    }
    for (const page of PAGE_HELP) {
      for (const id of page.terms ?? []) {
        expect(termIdSet.has(id), `${page.route} term ${id}`).toBe(true);
      }
      for (const id of page.topics ?? []) {
        expect(topicIdSet.has(id), `${page.route} topic ${id}`).toBe(true);
      }
    }
    for (const news of WHATS_NEW) {
      if (news.help) {
        expect(getEntry(news.help), `news ${news.id} help`).toBeDefined();
      }
    }
  });

  it("resolves every help: link and points every page link at a route", () => {
    for (const { from, body } of bodyList) {
      // A help: link the parser did not take (a bad id) would show as
      // literal text: the counts must agree.
      const writtenCount = body.split("](help:").length - 1;
      const linkIdList = helpLinkIdList(body);
      expect(linkIdList.length, `${from}: a malformed help: link`).toBe(
        writtenCount,
      );
      for (const id of linkIdList) {
        expect(getEntry(id), `${from} links help:${id}`).toBeDefined();
      }
      for (const to of routeLinkList(body)) {
        const path = to.split(/[?#]/)[0];
        expect(routeList, `${from} links ${to}`).toContain(path);
      }
    }
  });

  it("has page help for every route, and none for a route that is gone", () => {
    const helpRouteList = PAGE_HELP.map((page) => page.route);
    expect(new Set(helpRouteList).size, "a route has two entries").toBe(
      helpRouteList.length,
    );
    for (const route of routeList) {
      const covered =
        helpRouteList.includes(route) || PAGES_WITHOUT_HELP.includes(route);
      expect(covered, `no page help for route "${route}"`).toBe(true);
    }
    for (const route of [...helpRouteList, ...PAGES_WITHOUT_HELP]) {
      expect(routeList, `page help for unknown route "${route}"`).toContain(
        route,
      );
    }
  });

  it("knows every help id the UI uses", () => {
    const usedList: { id: string; file: string }[] = [];
    for (const [file, source] of sourceMap) {
      for (const pattern of USED_ID_PATTERN_LIST) {
        for (const match of source.matchAll(pattern)) {
          usedList.push({ id: match[1], file });
        }
      }
    }
    expect(usedList.length, "found no help id in the UI").toBeGreaterThan(0);
    for (const { id, file } of usedList) {
      expect(getEntry(id), `${file} uses unknown help id "${id}"`)
        .toBeDefined();
    }
  });

  it("points every tour step at a data-tour anchor that exists", () => {
    const anchorSet = new Set<string>();
    for (const source of sourceMap.values()) {
      for (const match of source.matchAll(TOUR_ANCHOR)) {
        anchorSet.add(match[1]);
      }
    }
    for (const tour of TOURS) {
      expect(routeList, `tour ${tour.id} route`).toContain(tour.route);
      expect(tour.steps.length, `tour ${tour.id} has no steps`)
        .toBeGreaterThan(0);
      for (const step of tour.steps) {
        expect(
          anchorSet.has(step.target),
          `tour ${tour.id}: no data-tour="${step.target}"`,
        ).toBe(true);
      }
    }
  });

  it("lists what's new newest first, with well-formed ids", () => {
    const idList = WHATS_NEW.map((news) => news.id);
    expect([...idList].sort().reverse()).toEqual(idList);
    expect(new Set(idList).size).toBe(idList.length);
    for (const news of WHATS_NEW) {
      expect(news.id, news.id).toMatch(/^\d{4}-\d{2}-\d{2}-\d+-[a-z0-9-]+$/);
      expect(news.id.startsWith(news.date), `${news.id} date`).toBe(true);
    }
  });

  it("suggests at least one topic to every audience", () => {
    for (const audience of HELP_AUDIENCE_LIST) {
      const count = TOPICS.filter((topic) =>
        topic.audience?.includes(audience),
      ).length;
      expect(count, `no topic for audience "${audience}"`).toBeGreaterThan(0);
    }
  });
});
