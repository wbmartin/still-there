# ProtoStack

The template other apps derive from: a Go stdlib
`net/http` + `pgx` API in `src/api-srv`, a React 19 / TypeScript / Vite /
Mantine web app in `src/web-app`. Nothing here is customer- or
app-specific — anything that becomes so belongs in the derived app, not
here.

## Layout

- `bin/` — numbered scripts: `1xx` build, `2xx` run, `3xx` tests, `4xx`
  certs, `5xx` db, `6xx` deploy.
- `etc/app.env` — the app's identity (`APP_SLUG`, `APP_DISPLAY_NAME`);
  every `bin/` script derives names from it. `DERIVING.md` + `bin/000-rebrand`
  turn this template into a new app.
- `etc/` — environment files. `uat-deploy.env` drives the presales
  `bin/600`/`610` pair (no inline defaults in the scripts).
- `infra/<host>/` — mirror of `/srv/<env>-proto/{bin,etc}` on a server.
  Scripts deploy verbatim (no secrets, no per-host values); every value
  comes from `etc/*.env`, whose repo copies are redacted `SERVERONLY`.
  Names derive from the app root's basename, so a job on both hosts is
  one file copied per host dir (`bin/300-run-tests infra` checks).
  Placeholder until post-sales; presales uses `bin/600`/`610`.
- `src/api-srv/sql/` — `migrations/NNNN_name.sql`, embedded in
  `cmd/migrate`. This is the **only** way a schema change reaches a
  database; never hand-run DDL. Also `roles.sql` and `fixtures/`.
- `doc/` — reference docs.
- `scripts/` — repo checks the gate calls: `check-doc-paths.sh` (every
  backticked repo path in a doc exists; `bin/300-run-tests docs`).

## Conventions

The rules that bite on every change. Long form: `doc/900-database-guidelines.md`
(schema), `doc/905-config-guidance.md` (env files), `doc/904-python-guidance.md`
(pytest and scripts).

- Schema: no foreign keys, no triggers, no `DEFAULT`s for application data
  (`create_ts`/`update_ts`/`delete_ts` are set in Go); every table carries
  `security_context_id`; primary key is `<table>_id`; snake_case.
- **`security_context_id` is the tenant. `1` is the first real tenant; `2` is
  the test tenant and is reserved — every automated test runs there and no
  real tenant is ever numbered 2.** A caller's tenant comes from the JWT, not
  from the request body (handlers overwrite what a client sends), so a test
  chooses its tenant by which seeded user it signs in as. Test databases hold
  no tenant 1: `sql/fixtures/test-tenant-lock.sql` puts
  `CHECK (security_context_id <> 1)` on every tenant table, and isolation is
  proved between 2 and 3. Fixtures, tests and derived apps keep this
  numbering.
- Security checks are called explicitly in each handler (JWT → rate limit →
  CSRF → authorization → `CheckAllChecksComplete`); exceptions are written in
  the handler, not hidden in middleware. Non-negotiable.
- A list is `fooList` / `foo_list`, never a plural — variables, functions
  (`getUserList`), endpoints (`/exampleList`). Verbose names over terse
  (`tagCache`, not `tc`); a variable holding an `ExampleRepository` is
  `exampleRepository` (Go receivers may be one letter).
- Go and TypeScript: break lines around 80 columns.
- Go gate (`bin/300-run-tests go`, every module): `gofmt -l` empty,
  `go vet` and pinned `staticcheck` with zero findings. Fix the code;
  `//lint:ignore <check> <reason>` only where the finding is wrong.
- Web gate (`bin/300-run-tests unit`): `eslint . --max-warnings 0` before
  vitest; check only, never `--fix`.
- A gate run leaves `git status` as it found it (checked at the end):
  test output goes under `tmp/` or is gitignored.
- Docs gate (`bin/300-run-tests docs`): a backticked repo path in a doc
  exists. A deleted file is named in prose without backticks; a real
  name outside the tree (bundle, host) goes in
  `scripts/doc-paths-allow.txt` with its reason; a placeholder is
  written `<entity>`.
- Web: call the API only through `apiJson` (`utils/api.ts`); `console.*`
  only in `utils/report-error.ts`; file names kebab-case; a file that
  mirrors a Go file names it in its header (`constants/api.ts`,
  `constants/grants.ts`).
- Go and web mirrors are hand-coded and drift-tested: Go writes a fixture,
  vitest replays it (`src/web-app/tests/contract.test.ts`;
  `bin/305-update-grants-fixture` after a grant change).
- Config is a flat `KEY=VALUE` file passed to the binary; no environment-
  variable lookups.
- Customer data never in git: source exports and anything derived from
  them live outside the repo; fixtures and mock data are synthetic
  (`DERIVING.md` section 2). A leak costs a new repository.
- Show only what the code verified: a count counts confirmed results, a
  metric is measured or absent, a missing value stays missing. Nothing
  fabricated reaches a screen, report or log.
- Git: summarize a PR from `git diff`, not memory, in markdown with headings.

## Branches

`main` is the default branch. Feature branches are `feature/<slug>`;
pull requests target `main`, and only Brandon advances `main` — agents
never merge into it without his approval, and touch only `main` and
their own branch.
