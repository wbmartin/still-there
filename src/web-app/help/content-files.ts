// The help content, gathered from one file per entry under help/content/
// (doc/151-help-module.md). One file each, never one shared list: two
// changes that each add help then never touch the same lines. Keyed by
// the file's path, so tests/help-content.test.ts can check every file is
// named for its entry. Vite (and vitest) resolve the globs at build time.
import type {
  HelpTerm,
  HelpTopic,
  PageHelp,
  WhatsNewEntry,
} from "./help-types";

type ByPath<T> = Record<string, { default: T }>;

export const GLOSSARY_FILE_MAP: ByPath<HelpTerm> = import.meta.glob(
  "./content/glossary/*.ts",
  { eager: true },
);

export const TOPIC_FILE_MAP: ByPath<HelpTopic> = import.meta.glob(
  "./content/topics/*.ts",
  { eager: true },
);

export const PAGE_FILE_MAP: ByPath<PageHelp> = import.meta.glob(
  "./content/pages/*.ts",
  { eager: true },
);

export const WHATS_NEW_FILE_MAP: ByPath<WhatsNewEntry> = import.meta.glob(
  "./content/whats-new/*.ts",
  { eager: true },
);

/** The entries of a file map, as an array. */
export const entryListOf = <T>(fileMap: ByPath<T>): T[] =>
  Object.values(fileMap).map((module) => module.default);

/** A file's base name: "./content/glossary/mfa.ts" is "mfa". */
export const fileBaseName = (path: string): string =>
  path.replace(/^.*\//, "").replace(/\.ts$/, "");
