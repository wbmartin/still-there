# Help module

An optional, dormant module: help built into the app. A Help panel on
every signed-in page explains the page the user is on, a glossary and
task articles answer "what is this / how do I", short tours point at the
parts of a page, and a usage log shows an admin where the help falls
short. It is in every app built from the template and shows nothing until
its config turns it on. The help itself is content written in the web
app, checked by tests like code, and stays authored while the module is
off.

## What it does

- **Help** in the header (and the **?** key anywhere outside a text box)
  opens a drawer on the current page's help: what the page is for, what to
  do there, its terms and topics, a **Show me around this page** tour when
  the page has one, **What's new**, every other topic and the glossary.
  The Help button carries a dot while there is a What's new entry the user
  has not seen; opening the panel marks it seen.
- **Search** in the panel ranks glossary terms and topics by title,
  alias, summary and body (`src/web-app/help/help-registry.ts`). A search
  that finds nothing is logged, so it can become new help.
- **Inline help** in pages: `<Term id>` (a dotted-underlined word whose
  definition shows on hover or keyboard focus), `<FieldHelp id>` (a "?"
  beside a label), `<HelpLink id>` (a link that opens the panel),
  `<Nudge id>` (a dismissible tip a page shows when its state calls for
  one). With help off they are plain text or nothing.
- **Tours**: short step-by-step cards anchored to `data-tour="<name>"`
  attributes in a page; keyboard operable (arrows, Escape); replayed from
  the panel, or started once on a user's first visit when the tour says
  `autoStart`.
- **Cmd+K / Ctrl+K** opens a command palette (`@mantine/spotlight`): the
  pages of the app's navigation the user's grants and the modules on
  allow (`src/web-app/constants/nav.ts`, the navbar's list too: an item
  may name a grant and a module flag from `/me`, as Issue reports does)
  and every help entry.
- **Deep links**: `?help=<id>` on any signed-in address opens the panel
  on that entry and then drops the parameter, for links in emails, chat
  or error messages.
- **Help insights** (`/helpInsights`, linked from the panel and the
  palette for a user admin): the tenant's searches with no result, the
  most-read help, and tours finished and skipped, over 7, 30 or 90 days.

What each user has seen (the newest What's new id) and finished (tours,
dismissed nudges) is kept in their own preferences document
(`app_user.preference_jsonb`, under the key `help`), so it follows them
from browser to browser; there is no browser storage. The Profile page's
save carries that key through (`carriedPreference` in
`src/web-app/types/profile-type.ts`).

## Turning it on

`HELP_ENABLED=true` in the API's env file (`doc/905-config-guidance.md`),
then restart the API. The key is optional: absent or `false` is off, and
any other value stops start-up. The server logs `HELP_MODULE` at start
with the state. `GET /me` answers `helpEnabled`, and the web app shows
the Help button, the ? key, the palette and the inline help only when it
is true. While off, `POST /helpEvent` and `GET /helpInsights` answer 404
after their security checks, and the insights page says help is off.

The e2e API (`etc/e2e-api-srv.env`) runs with it on, so the Playwright
spec `src/web-app/tests/e2e/help.spec.ts`, the route sweep and pytest
`tests/test_help.py` reach it. `etc/loc-api-srv.env` has it off; the host
files carry the key commented out.

## Authoring help

Help is typed data in `src/web-app/help/content/`, written in the same
change as the feature it explains. **Every new page gets a page entry in
the same PR**: the content tests fail while a route in
`src/web-app/constants/routes.ts` has none.

| File | Holds | Exported as |
|------|-------|-------------|
| `src/web-app/help/content/pages.ts` | one entry per route | `PAGE_HELP: PageHelp[]`, `PAGES_WITHOUT_HELP: string[]` |
| `src/web-app/help/content/glossary.ts` | the app's words | `GLOSSARY: HelpTerm[]` |
| `src/web-app/help/content/topics.ts` | task articles | `TOPICS: HelpTopic[]` |
| `src/web-app/help/content/tours.ts` | page tours | `TOURS: Tour[]` |
| `src/web-app/help/content/whats-new.ts` | release notes, newest first | `WHATS_NEW: WhatsNewEntry[]` |
| `src/web-app/help/content/audience.ts` | who help is for, by grant | `HELP_AUDIENCE_LIST`, `DEFAULT_HELP_AUDIENCE`, `HELP_AUDIENCE_BY_GRANT` |

The types are in `src/web-app/help/help-types.ts`:

- `PageHelp`: `route` (a `ROUTES` value), `title`, `purpose` (one or two
  sentences), optional `steps` (what to do here, a sentence each),
  `terms` (glossary ids), `topics` (topic ids).
- `HelpTerm`: `id`, `term`, `short` (the hover text: one sentence, at
  most 30 words), `body`, optional `aliases` (other words people search
  by), `related` (term or topic ids), `audience`.
- `HelpTopic`: `id`, `title`, `summary` (at most 35 words), `body`,
  optional `terms` (glossary ids), `related`, `audience`.
- `Tour`: `id`, `route`, `title`, optional `autoStart`, `steps` of
  `TourStep`: `target` (the `data-tour` value of the element it points
  at), `title`, `body` (plain text, one or two sentences).
- `WhatsNewEntry`: `id` (`YYYY-MM-DD-N-slug`, N counting up within a day,
  so ids sort newest first), `date` (`YYYY-MM-DD`, the id's prefix),
  `title`, `body`, optional `help` (the entry "Learn more" opens).

Ids (terms, topics, tours) are lower-case kebab, at most 64 characters,
and a term never shares an id with a topic. An `audience` value is one of
`HELP_AUDIENCE_LIST`; the template has one, `everyone`. A derived app
with distinct kinds of user adds its audiences and maps grants to them
(the first grant held wins); an audience only orders help under
"Suggested for you" and never hides any.

A `body` is a small markdown subset (`src/web-app/help/help-markdown.ts`),
rendered as React elements, never as HTML:

- a blank line starts a new block; a paragraph's lines are joined;
- every line starting `- ` is a bulleted list, `1. ` a numbered one; an
  indented line continues the item above it (to wrap at 80 columns);
- `**bold**`;
- `[label](help:<id>)` opens another entry in the panel;
- `[label](/route)` links to a page of the app.

Anything else, raw HTML and links off the app included, shows as text.

In a page, use the components with a literal id, so the tests can read
it: `<Term id="<term>">word</Term>`, `<FieldHelp id="<term-or-topic>" />`,
`<HelpLink id="<term-or-topic>">text</HelpLink>`,
`<Nudge id="<tip>" help="<term-or-topic>" title="...">text</Nudge>`, and
`data-tour="<target>"` on what a tour points at. For a one-line hint with
no entry behind it, `src/web-app/components/help-tip.tsx` stays the
lighter choice.

### What the content tests check

`src/web-app/tests/help-content.test.ts` runs in `bin/300-run-tests unit`,
with the module on or off:

- ids are well formed and unique across terms and topics; tour ids unique;
- `short` at most 30 words, `summary` at most 35, no empty body;
- every `related`, `terms`, `topics` and What's new `help` reference
  resolves;
- every `help:` link resolves and parsed as a link, and every
  `[label](/route)` names a route in `ROUTES`;
- every route in `ROUTES` has a page entry or is in `PAGES_WITHOUT_HELP`,
  and no entry names a route that is gone;
- every id the UI uses (`<Term|FieldHelp|HelpLink id=...>`,
  `<Nudge help=...>`, `openHelp("...")` in `pages/`, `components/`,
  `help/` and `app.tsx`) exists;
- every tour's route is a route and each step's target is written as
  `data-tour="..."` somewhere in that source;
- What's new is newest first, ids well formed and unique;
- every audience has at least one topic.

`src/web-app/tests/help-markdown.test.tsx` pins the markdown subset and
that a `<script>` in a body stays text; `help-registry.test.ts`,
`help-preference.test.ts` and `help-panel.test.tsx` cover search, the
saved state and the panel.

## Insights

`POST /helpEvent` (any signed-in user, `LOGON`) records one of
`search_no_result` (the text typed, cut to 200 characters),
`entry_view` (a term or topic id), `tour_complete` or `tour_skip` (a tour
id), with the route the panel was on, in the caller's session tenant
(`help_event`, migration `src/api-srv/sql/migrations/0007_help_event.sql`).
It is fire-and-forget from the web app; each user's events past 500 in a
rolling day are dropped and the answer says `recorded: false`.

`GET /helpInsights?days=N` (1 to 365, default 30) counts the session
tenant's events by search text, entry and tour, at most 25 of each, most
frequent first. It names no one. It needs `READ_USER` (9002), the user
admin's read grant, because search text is whatever users typed and may
hold anything. Use it to decide what to write next: a search with no
result wants a term, an alias or a topic.

The migration is 0007, after the issue-report module's 0006
(`doc/150-issue-report-module.md`): this module is stacked on that one.

## Export to a host

`bin/620-export-tenant-data` does not carry help usage: its scrub
(`src/api-srv/sql/export-scrub.sql`) removes every `help_event` row from
the copy it dumps. They are the laptop's own searches and views, which
would skew the host's Help insights, and search text is whatever was
typed there. The help content itself is code and ships with the build.

## Where the code is

- Web, framework: `src/web-app/help/` (`help-registry.ts`,
  `help-markdown.ts`, `help-preference.ts`, `help-panel.tsx`,
  `tour-runner.tsx`, `command-palette.tsx`, `help-shell.tsx`, the inline
  components), `src/web-app/stores/help-store.ts`,
  `src/web-app/pages/help-insights-page.tsx`. `help-shell.tsx` (panel,
  tours, palette, the ? key, deep links) and the insights page are lazy
  chunks, loaded only while signed in with help on.
- Web, content: `src/web-app/help/content/`.
- API: `src/api-srv/internal/handler/help_handler.go`,
  `src/api-srv/internal/model/help_model.go`,
  `src/api-srv/internal/repository/help_event_repository.go`; the flag in
  `src/api-srv/internal/config/config.go`.

## Not in it yet: asking a question

An assistant that answers a typed question from the help content is a
later phase. Its seam is `HelpPanel`'s optional `askSlot` prop
(`src/web-app/help/help-panel.tsx`), which renders under the search box
while the user types; the template passes none, so there is no Ask UI.

## Removing it

Leave `HELP_ENABLED` unset: nothing shows and the routes answer 404. To
take it out of a derived app's code: drop the `<HelpButton>` and
`<HelpShell>` lines and the insights route from `src/web-app/app.tsx`,
the inline components from pages, `src/web-app/help/`, the help store,
the insights page and their tests, the two routes in
`src/api-srv/cmd/server/main.go` with their handler, model and
repository, and add a migration that drops `help_event`.
