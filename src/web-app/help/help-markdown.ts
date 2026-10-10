// The help content's markdown subset, parsed to data that
// help-markdown-view.tsx renders as React elements (doc/151-help-module.md).
// Never HTML: a `<script>` in a body is text on the screen, not markup.
//
//   blank line          a new block
//   "- " on each line   a bulleted list
//   "1. " on each line  a numbered list
//   anything else       a paragraph (its lines joined by spaces)
//   an indented line    continues the line before (wraps a list item)
//   **bold**            bold
//   [label](help:<id>)  opens that help entry in the panel
//   [label](/route)     a link to a page of this app
//
// A link to anything else (http:, //host, javascript:) is not a link: it
// stays as its literal text, so content cannot send a user off the app.

export type HelpInline =
  | { kind: "text"; text: string }
  | { kind: "bold"; text: string }
  | { kind: "help-link"; label: string; id: string }
  | { kind: "route-link"; label: string; to: string };

export type HelpBlock =
  | { kind: "paragraph"; inlineList: HelpInline[] }
  | { kind: "bullet-list"; itemList: HelpInline[][] }
  | { kind: "number-list"; itemList: HelpInline[][] };

// A help id: lower-case letters, digits and hyphens (the server's shape).
const HELP_ID = "[a-z0-9][a-z0-9-]{0,63}";
// An app path: one leading slash, never two (//host is another site).
const ROUTE_PATH = "/(?!/)[^)\\s]*";

const INLINE_PATTERN = new RegExp(
  `\\*\\*([^*]+)\\*\\*|\\[([^\\]]+)\\]\\((help:${HELP_ID}|${ROUTE_PATH})\\)`,
  "g",
);

export function parseHelpInline(text: string): HelpInline[] {
  const inlineList: HelpInline[] = [];
  let last = 0;
  for (const match of text.matchAll(INLINE_PATTERN)) {
    const at = match.index;
    if (at > last) {
      inlineList.push({ kind: "text", text: text.slice(last, at) });
    }
    if (match[1] !== undefined) {
      inlineList.push({ kind: "bold", text: match[1] });
    } else if (match[3].startsWith("help:")) {
      inlineList.push({
        kind: "help-link",
        label: match[2],
        id: match[3].slice("help:".length),
      });
    } else {
      inlineList.push({ kind: "route-link", label: match[2], to: match[3] });
    }
    last = at + match[0].length;
  }
  if (last < text.length) {
    inlineList.push({ kind: "text", text: text.slice(last) });
  }
  return inlineList;
}

// A block's lines, with an indented line joined to the one before it: a
// long list item wraps onto lines that start with spaces.
function logicalLineList(block: string): string[] {
  const lineList: string[] = [];
  for (const rawLine of block.split("\n")) {
    const line = rawLine.trim();
    if (/^\s/.test(rawLine) && lineList.length > 0) {
      lineList[lineList.length - 1] += ` ${line}`;
    } else {
      lineList.push(line);
    }
  }
  return lineList;
}

export function parseHelpMarkdown(body: string): HelpBlock[] {
  const trimmed = body.trim();
  if (!trimmed) return [];
  return trimmed.split(/\n\s*\n/).map((block): HelpBlock => {
    const lineList = logicalLineList(block.trim());
    if (lineList.every((line) => line.startsWith("- "))) {
      return {
        kind: "bullet-list",
        itemList: lineList.map((line) => parseHelpInline(line.slice(2))),
      };
    }
    if (lineList.every((line) => /^\d+\. /.test(line))) {
      return {
        kind: "number-list",
        itemList: lineList.map((line) =>
          parseHelpInline(line.replace(/^\d+\. /, "")),
        ),
      };
    }
    return {
      kind: "paragraph",
      inlineList: parseHelpInline(lineList.join(" ")),
    };
  });
}

// Every inline of a body, flat: what the content gates walk.
function inlineListOf(body: string): HelpInline[] {
  return parseHelpMarkdown(body).flatMap((block) =>
    block.kind === "paragraph" ? block.inlineList : block.itemList.flat(),
  );
}

/** The ids a body's [label](help:<id>) links point at. */
export const helpLinkIdList = (body: string): string[] =>
  inlineListOf(body).flatMap((inline) =>
    inline.kind === "help-link" ? [inline.id] : [],
  );

/** The paths a body's [label](/route) links point at. */
export const routeLinkList = (body: string): string[] =>
  inlineListOf(body).flatMap((inline) =>
    inline.kind === "route-link" ? [inline.to] : [],
  );
