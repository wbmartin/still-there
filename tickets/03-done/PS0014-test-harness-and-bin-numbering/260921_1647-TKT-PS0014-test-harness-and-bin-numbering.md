# 2026-09-21 — PS0014 one gate

`bin/300-run-tests` is the whole test story now: go, vitest (which had 28
tests nobody ran), playwright, pytest, and `load` on request. The two
"test" scripts that rewrote sources are gone. Reports and logs live under
`tmp/`; `tests/reports/` is history.

Two small design points: the double e2e reseed is fixed with a single
env flag (`E2E_DB_SEEDED`) rather than by taking the reseed out of
Playwright's global setup, so a bare `npx playwright test` stays
self-contained; and the ports/passwords are derived from one place each
(playwright reads the API's env file; the test env files build their
URLs from `PG*`).

The load tester moved out of the API module into its own — stdlib only,
so the go.mod is three lines — and lost its always-on
`InsecureSkipVerify` and compiled-in password. A subagent wrote 50 Go
unit tests for the pure packages; I spot-checked the jwt and CORS ones.
Gate: go=163 unit=28 e2e=3 py=63.
