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
claimed: 2026-09-21 by TKT-PS0014-test-harness-and-bin-numbering
merged: 6c0cbe9
deployed: none
validated: 2026-09-21
---
# PS0014 — Four-and-a-half test harnesses and two generations of bin/ scripts

A template review found five overlapping test harnesses (Go, pytest,
Playwright, vitest — never wired into the gate — and a load-test tool),
bin/ scripts split across two numbering eras with a double e2e database
reseed, hardcoded ports/passwords, and test scripts that mutated source
files. Brandon confirmed pytest stays as the black-box API/E2E suite; the
ticket executed its own bin/ layout proposal and closed the coverage gaps
it could.

## What changed
- `bin/300-run-tests [all|go|unit|e2e|py|load]` replaces `200-run-tests`:
  `unit` runs vitest (28 tests, previously never run by any gate); `py`
  bootstraps its own venv and runs pytest directly (`320` retired);
  `load` calls the load tester (not part of `all`).
- `bin/110-build-web-app` (from `222`): writes `dist/<slug>-web_<ts>.tar.gz`
  (was the repo root); follows the env-file contract.
- `bin/515-reseed-db [name]` (from `514-reset-e2e-fixtures`): truncate +
  fixtures for any scratch db, falls back to `510`; `513` deleted.
- `bin/400-make-local-dev-certs` (new): self-signed `etc/crt/api-srv-*.pem`,
  `chmod 600`, refuses to overwrite without `FORCE=1`.
- `bin/330-test-load`: env contract (`TEST_BASE_URL` dies by name, no
  silent fallback), health check before running, report to
  `tmp/test-reports/`.
- `bin/310-test-repo-model`, `bin/320-test-web-service` deleted — both
  mutated the source tree (`golines -w`, `eslint --fix`); `git status` is
  clean after `bin/300 all`.
- Single e2e reseed per run: `300 e2e` exports `E2E_DB_SEEDED=1`,
  `tests/e2e/global-setup.ts` skips its own reseed when set; a bare
  `npx playwright test` still reseeds itself.
- Ports/passwords derived, not repeated: `playwright.config.ts` reads
  `SERVER_PORT`/`SERVER_BIND_ADDRESS` from the e2e API env file;
  `etc/loc-test.env` and `infra/uat/etc/uat-proto-test.env` build
  `TEST_PG_URL_BASE`/`TEST_DATABASE_URL` from
  `PGUSER/PGPASSWORD/PGHOST/PGPORT` (password written once, not three
  times).
- `cmd/load-test` moved to `src/load-test/`, its own stdlib-only Go
  module; `--user`/`--password` flags replace the compiled-in
  `admin/admin123!`; `--insecure` (default false) replaces the
  unconditional `InsecureSkipVerify`.
- 50 new Go unit tests (subagent-written, spot-checked): `auth/jwt_test.go`,
  `auth/password_test.go`, `auth/rbac_test.go`,
  `security/security_headers_test.go`, `repository/scoped_query_test.go`.
- pytest hygiene: `pytest 7.4.3→8.4.2`, `requests 2.31.0→2.32.5` (CVE-2024-
  35195), `psycopg2-binary` removed (unused); dead `/auth2` branch in
  `conftest.py` replaced with a `pytest.fail` naming the actual
  `mfa_exempt` requirement.
- Docs swept for retired script names (`README.md`, `DERIVING.md`,
  `CLAUDE.md`, `doc/902`, `doc/330`, `tests/README.md`, `infra/README.md`,
  `ticket-workflow` skill).

## Rules that survived
- `bin/300-run-tests` is the single test gate; no test script writes to
  the source tree.
- One e2e database reseed per `bin/300 e2e` run, via `E2E_DB_SEEDED`, not
  by removing Playwright's own global-setup reseed.
- Ports and passwords are derived from one env source each, never
  repeated by hand across files.
- Go carries the unit-test burden; handlers stay thin (PS0009) so the
  logic they call is testable without `httptest`; pytest stays the
  black-box/E2E suite by Brandon's direction, not ported to Go.
- `cmd/` holds only shipped binaries — tooling like the load tester lives
  in its own module.

## Figures at close
| | date |
|---|---:|
| go tests | 163 (was 113) |
| vitest unit tests | 28 (newly wired into gate) |
| e2e tests | 3 |
| pytest | 63 |
| new Go unit tests added | 50 |

## Endpoints
None.

## Security
Profiles 7 (dependencies) and 8 (scripts/config). The `requests` upgrade
closes CVE-2024-35195; `psycopg2-binary` removal drops an unused native
wheel from every test host. The load tester's `InsecureSkipVerify` and
compiled-in admin password are now opt-in flags rather than defaults, so
pointing it at a real host verifies the certificate and a rotated-fixture
host isn't sprayed with a stale credential; `330` refuses to run unless
`/health` answers first. No test script mutates the source tree any more,
so a green gate can't be one that silently reformatted the code it
tested. `vite.config.ts` now fails loudly without `VITE_API_PROXY_URL`
instead of silently proxying to `localhost:8080`.

## Left open
- Web unit coverage gaps (page/component tests, `use-auth-guard`, store
  extras) listed in the ticket, left for PS0012/PS0013 — now counted
  since vitest is in the gate.
- pytest vs. Go for black-box coverage remains a conscious, revisitable
  choice, not a closed decision.

## Sources
- `tickets/04-archive/engineering-log/260921_1647-TKT-PS0014-test-harness-and-bin-numbering.md`
- git: `6c0cbe9`
