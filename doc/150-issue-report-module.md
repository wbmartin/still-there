# Issue-report module

An optional, dormant module: the people using an app report bugs, polish
and feature requests from inside it, and follow the replies there. It is
in every app built from the template and does nothing until its config
turns it on.

## What it does

- A **Report** button in the header, on every signed-in page, opens a
  drawer: Bug / Polish / Feature, a one-line summary, a markdown
  description (Write / Preview), and up to five files. **Capture screen**
  adds a PNG of the current page (`html-to-image`: only this tab, no
  browser prompt; the drawer and notifications are left out, and so is
  anything marked `data-capture-ignore`). The page's address, the browser
  and the viewport go along to help reproduce it.
- **Issue reports** (`/issueReportList`, in the nav) lists the tenant's
  reports, open / closed / all, and shows one with its files and its
  conversation. A reporter replies with markdown and files, **withdraws** a
  report that is not closed and **reopens** one that is. Staff of the
  tracker own every other status.
- Every signed-in user (grant `LOGON`) files reports and sees all of the
  tenant's reports; another tenant's are 404, like any record.
- Reports, messages and files are stored in the app first (migration
  `src/api-srv/sql/migrations/0006_issue_report.sql`: `issue_report`,
  `issue_report_message`, `issue_report_attachment`; files as `bytea`). A
  background **relay** (`src/api-srv/internal/issuerelay/`) sends them to
  an issue tracker and, every interval, pulls back each open report's
  status and staff replies. With no relay configured, reports stay in the
  app (status "Received").

Files follow doc/105, Serving user files: the type is read from the bytes
(an image extension must hold PNG, JPEG, GIF or WebP; .pdf a PDF;
.txt, .log and .csv plain text; anything else is 415), 10 MB each,
18 MB a request; on the way out the bytes are sniffed again, only an image
is shown inline, nothing is cached, and HTML is always a download as
`application/octet-stream`. An .html page (a mock-up of the change) may
go on a polish or feature report, never on a bug.

## Turning it on

Config keys, in the API's flat config file (`doc/905-config-guidance.md`).
All are optional: a file without them leaves the module off, and the API
logs `Issue reports off` and `Issue relay off` at start.

| Key | Value |
|---|---|
| `ISSUE_REPORTS_ENABLED` | `true` turns the module on: the Report button, the page, the `/issueReport*` routes. Absent or `false`: no button, and every route answers 404 after its security checks |
| `ISSUE_RELAY_URL` | the tracker's base URL, `https://...` |
| `ISSUE_RELAY_PROJECT_ID` | the project's id in the tracker |
| `ISSUE_RELAY_TOKEN` | the project's report token: a secret, in the host's file only |
| `ISSUE_RELAY_INTERVAL_SECONDS` | how often the relay sends and syncs; default 120, at least 15 |

The three `ISSUE_RELAY_*` target keys go together: all three set turn the
relay on (`Issue relay on, every 120s` in the log), none leaves it off, and
one or two set stop the API at start, naming the missing ones. Reports
filed while the relay was off stay in the app; reports filed after are
sent on the next tick, and a failed send is retried up to 20 times (shown
as "Not delivered yet").

Reports filed while the relay was off (status "Received", relay state
`local`) are sent when an API starts with the relay on: at start the relay
queues every live, open `local` report, oldest first, and every `local`
message on a report it sends (`Issue relay: N local reports and M local
messages queued` in the log). A message still goes only after its report
is in the tracker. A report withdrawn while the relay was off stays in
the app with its messages: sending it would only announce something
already closed.

No repository env file carries a tracker's address or token
(`TestLoadConfig_RepoEnvFiles`). Setting the keys on a host is the derived
app's own step: add them to that host's API env file and restart the API.

## The tracker side (Radley House)

In the House: the project → **In-app reports** → **Report token** mints
the token (shown once). The relay calls the House's report API under
`{ISSUE_RELAY_URL}/api/projects/{ISSUE_RELAY_PROJECT_ID}/external/reports`
with the token in the `X-Report-Token` header:

| Call | What |
|---|---|
| `POST` (multipart) | file a report: `kind`, `summary`, `detail`, `reporter_name`, `reporter_email`, `reporter_ref` (the app's user id), `workspace`, `context` (JSON), `files` (up to 5) |
| `GET ?ids=a,b` | status and conversation of those reports |
| `POST /{id}/messages` (multipart) | a reporter's reply: `body`, `author_name`, `author_ref`, `files` |
| `POST /{id}/withdraw`, `/{id}/reopen` | the reporter's actions, with an optional `{"note"}`; a 409 means the tracker is already there and counts as done |
| `GET /{id}/attachments/{aid}` | a staff reply's file, fetched when a user opens it |

`workspace` is the report's tenant, sent as `tenant <id>`. A derived app
with tenant names sets `Relay.WorkspaceLabel` in `src/api-srv/cmd/server/main.go`.

## What leaves the app

With the relay on, each report goes to the tracker with its files (a
screen capture shows whatever was on the reporter's screen), the
reporter's name, email and user id, and the browser context. The drawer
says that the page and browser go along and that the workspace sees the
report. Only the server talks to the tracker, over the configured URL,
with a token that opens the report routes of that one project and nothing
else; a tracker that is down only delays sending (reports wait in the
app). Staff replies and their files come back the other way; a staff
file is fetched from the tracker when opened and served under the same
rules as an uploaded one.

## Export to a host

`bin/620-export-tenant-data` does not carry issue reports: its scrub
(`src/api-srv/sql/export-scrub.sql`) removes the three tables' rows from
the copy it dumps. They are the laptop's conversation with a tracker, and
their tracker ids would be synced against the host's own project.

## Where the code is

- API: `src/api-srv/internal/handler/issue_report_handler.go`,
  `src/api-srv/internal/repository/issue_report_repository.go`,
  `src/api-srv/internal/model/issue_report_model.go`,
  `src/api-srv/internal/issuerelay/`; config in
  `src/api-srv/internal/config/config.go`; `/me` carries
  `issueReportsEnabled` for the web app.
- Web: `src/web-app/components/issue-report-drawer.tsx`,
  `src/web-app/components/issue-attachment-picker.tsx`,
  `src/web-app/components/markdown-editor.tsx`,
  `src/web-app/components/markdown-view.tsx`,
  `src/web-app/stores/issue-report-store.ts`,
  `src/web-app/pages/issue-report-list-page.tsx`. The page and the drawer
  load on first use, so an app with the module off never fetches them.
- Tests: `issue_report_handler_test.go`, `issue_report_repository_test.go`
  (with the cross-tenant test), `src/api-srv/internal/issuerelay/relay_test.go` (a fake tracker),
  `src/web-app/tests/issue-report-store.test.ts`,
  `src/web-app/tests/issue-report-drawer.test.tsx`,
  `src/web-app/tests/e2e/issue-report.spec.ts`,
  `tests/test_issue_report.py`. The e2e and pytest API runs with the
  module on (`etc/e2e-api-srv.env`) and no relay.

## Removing it

A derived app that does not want the module leaves it off. To take it out
entirely: delete the files above, the six routes in `main.go` and their
rows in `route_check_contract_test.go`, the `ISSUE_*` config fields, the
`issueReportsEnabled` field of `/me`, and the header button, nav link and
route in `src/web-app/app.tsx`; drop the tables with a new migration.
