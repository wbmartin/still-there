---
id: PS0014
title: Test harnesses, bin/ numbering and env-file contract, tmp/ conventions, coverage gaps
status: ready
blocked_by:
waiting_on:
priority: medium
domain: tooling
raised: 2026-09-21
raised_by: template review 2026-09-21
asked:
claimed:
merged:
deployed:
validated:
---
# PS0014 — Four-and-a-half test harnesses and two generations of bin/ scripts

**Source:** template review 2026-09-21. **Decision ticket** for pytest's
fate; the rest is execution.

## Test harnesses today

| Harness | Where | Size | Run by |
|---|---|---|---|
| Go `go test` | `src/api-srv` | 9 `_test.go`: 4 repository (integration, skip without DB), 2 model (trivial), 1 handler (user only, mocked), migrate (8 unit + DB), kv_memory | `bin/200 go`, `bin/310` |
| pytest HTTP suite | `tests/` | 8 files, ~60 tests | `bin/200 py`, `bin/320` |
| Playwright | `src/web-app/tests/e2e` | 1 spec, 3 smoke tests | `bin/200 e2e`, `bin/230` |
| vitest | `src/web-app/tests` | 3 files / 22 tests (api classification, auth-store refresh, entity-store cache) | **not run by `bin/200`** |
| Go load-test | `src/api-srv/cmd/load-test` | 1,130 lines incl. inline HTML report, `InsecureSkipVerify`, hardcoded `admin/admin123!` (`client.go:62-63`) | `bin/330` |

**pytest is not redundant today:** it is the only black-box coverage of
auth, CSRF, rate limiting, 409/410 semantics and security-context
isolation (Go handler tests cover the user handler only; Playwright is
3 smoke tests). But it drags in a third language, a venv, `bin/320`'s
venv-rebuild logic, `psycopg2-binary` (imported in `conftest.py`, never
used), and 2023-era pins (`pytest==7.4.3`, `requests==2.31.0` — CVE
2024-35195 fixed in 2.32). `tests/README.md` is stale on every
operational point (PS0006 table).

Options: (a) promote pytest to the canonical API contract suite, say so
in README, bump pins, drop psycopg2, rewrite `tests/README.md`; (b) port
it to Go `httptest` or Playwright `request` fixtures over time and retire
it. PS0001 chose "keep pytest"; revisit consciously.

## Coverage gaps (regardless of harness decision)

- Go: none for `config.go`, `auth.go` (JWT/argon2), `app_helper.go` (all
  security checks), `rbac.go`, `security_headers.go`, `kv_client.go`,
  `logging_client.go`, `scoped_query.go`, `retry.go`, `auth_handler.go`,
  `user_mfa_handler.go`, `example_handler.go`, `app_user_group_handler.go`,
  `cmd/*`. Priority: `buildScopedWhere` (pure), the middleware (PS0009),
  the example handler (it is the template).
- Web: 0 page/component tests; `use-auth-guard.ts` untested;
  `app-user-store` extras (mergeUserList, group add/remove, readUserById),
  `examples-store.readExampleById`, `grants.ts` helpers, ChangePassword
  validation, MFA flow, `cache-utils`. `tests/e2e/fixtures/api-client.ts`
  is built for per-entity CRUD specs and unused.

## bin/ numbering and contract

Scheme per CLAUDE.md: `1xx` build, `2xx` run, `3xx` test, `4xx` certs,
`5xx` db, `6xx` deploy. Drift:

- `010-download-shrmpl` — no band (PS0003 decides its fate).
- `200-run-tests` is the test gate but sits in the run band; `310/320/330`
  are the "test" band; `230-run-web-app-e2e` is a test in the run band.
- `222-build-web-app` is a build in the run band and duplicates step 3 of
  `bin/600` with a different archive name, written to the **repo root**
  (not `dist/`, not gitignored).
- `310-test-repo-model` runs `golines -m 80 -w .` (reformats sources,
  requires golines on PATH) and `320-test-web-service` runs `eslint --fix`
  — "test" scripts mutate the tree. `310` is a subset of `200 go` plus
  side effects.
- `510-reset-test-db`, `513-reset-e2e-db` (thin wrapper), `514-reset-e2e-fixtures`
  (truncate-not-drop, exists because Playwright holds a pool connection);
  `511/512` don't exist. `bin/200 e2e` calls 513 **and then**
  `tests/e2e/global-setup.ts` calls 514 via a
  `../../../../bin/514-reset-e2e-fixtures` relative path — double reset
  every run.
- `4xx` has one script (shrmpl mTLS). API-server TLS certs
  (`etc/crt/api-srv-*.pem`, referenced by `loc-api-srv.env` `TLS_*`) have
  no generator; `doc/310-api-srv-cert-renewal.md` is prose.
- Sep-2026 scripts (`200`, `510-514`, `600`, `610`) follow the env-file
  contract (source `etc/*.env`, missing keys die by name, no inline
  defaults). Dec-2025 scripts (`215`, `220`, `222`, `310`, `320`, `330`,
  `410`) do not: `330` has a `|| echo http://localhost:8080` fallback that
  never fires (pipeline exit status); `215` daemon mode writes
  `tmp/server.log|pid` relative to CWD with no `mkdir -p`.
- Ports are literal in 21 files; `playwright.config.ts` (`E2E_WEB_PORT=5182`,
  `E2E_API_PORT=8082`) and `etc/e2e-api-srv.env` `SERVER_PORT=8082` must
  agree by hand. → playwright reads them from `PROTO_E2E_API_ENV`.
- `etc/loc-test.env` and `infra/uat/etc/uat-proto-test.env` repeat the
  same password three times each (`PGPASSWORD`, `PROTO_PG_URL_BASE`,
  `PROTO_TEST_DATABASE_URL`). → derive URLs in `bin/510` from
  `PGUSER/PGPASSWORD/PGHOST`.

## tmp/ and output conventions diverge

`215` → `tmp/server.*` (CWD-relative); `200` → `tmp/test-logs/`;
`310/320/330` → `tests/reports/` (tracked — PS0002 untracks); `222` →
tarball in repo root; `600` → `dist/`;
`etc/loc-shrmpl-log-srv.env` → absolute `/Users/brandon/…/tmp/log`.
→ everything transient under `$PROJECT_ROOT/tmp/<script>/` or `dist/`,
both gitignored.

## Proposed bin/ layout

```
100-build-api-server   110-build-web-app (from 222; writes dist/)
215-run-api-srv        220-run-web-app        (210/211 only if PS0003 keeps shrmpl, renamed 9xx)
300-run-tests [all|go|unit|e2e|py|load]   ← today's 200, absorbs 320's venv bootstrap and runs vitest
330-test-load          (keeps 330 or becomes a 300 mode)
400-make-local-dev-certs  (api-srv TLS)   410-… only with shrmpl
510-reset-test-db [name]   515-reseed-db [name]  (truncate + fixtures; replaces 513/514)
600-bundle-uat  610-deploy-uat
```

`cmd/load-test` moves to `tools/loadtest` or its own module, or is
replaced by k6/hey invoked from `330` (PS0015 lists it too — one of the
two tickets takes it).

## From the PS0006 triage of doc/000-TODO.md (2026-09-21)

- Handler tests never go through HTTP: `MockUserRepository` exists but
  `TestRetrieveUserList` calls the mock directly, because `NewAppHelper`
  needs live `AppServices`. Decide whether handlers get an HTTP-level unit
  test (a test `AppServices` with the in-process KV and a buffer logger is
  cheap after PS0003) or stay covered by pytest only.

## Acceptance

- `bin/300-run-tests all` runs Go + vitest + Playwright (+ pytest if kept)
  and is the only gate; no test script mutates sources.
- Every `bin/` script sources its env file and dies by key name; no
  `localhost:` literal outside `etc/`.
- One e2e reseed per run.
- `git status` clean after `bin/300 all`.
- `doc/902-project-commands.md`, `README.md` key-commands table,
  `ticket-workflow` skill check step, `tests/README.md` all match.

## Deploy needs

None. `deployed: none`.

## Direction (Brandon, 2026-09-21)

"Seems fine." Proceed. **pytest stays** for black-box / e2e API tests
(Brandon, 2026-09-21: "pytest is fine… I've got python for the e2e
tests"). His stated preference for the *unit* level: tests in Go, with
the mockable logic separated from the HTTP layer that calls it — i.e.
handlers stay thin and explicit (PS0009), the functions they call are
testable without `httptest`. Apply that when adding Go tests here; do
not port the pytest suite to Go. Item "cmd/load-test out of cmd/" is taken
by this ticket, not PS0015.
