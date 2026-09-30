---
id: PS0022
title: Move every automated test into tenant 2 (security_context_id 2 is the reserved test tenant)
status: ready
blocked_by:
waiting_on:
priority: medium
domain: data
raised: 2026-09-30
raised_by: Brandon
asked:
claimed:
merged:
deployed:
validated:
---
# PS0022 — Tests run in tenant 2

**Decision (Brandon, 2026-09-30):** `security_context_id` 1 is the first
real tenant; **2 is the test tenant and is reserved.** Every automated
test runs there and no real tenant is ever numbered 2. The rule is
already recorded in `CLAUDE.md` (Conventions), `doc/900-database-guidelines.md`
and the `local-seed.sql` header; this ticket moves the tests to match.

## Why it is cheap in principle

A caller's tenant comes from the JWT claim, not the request body — every
handler overwrites a client-supplied `securityContext` (proved today by
`tests/test_security_context_isolation.py`). So a test picks its tenant
by **which seeded user it signs in as**, not by what it posts.

## What has to change

- **`sql/fixtures/local-seed.sql`** — the work. Today `admin`, `basic` and
  `mfa` are all in context 1; context 2 has groups 4 (`Basic2 User`) and 5
  (`Admin2`) and no users. Seed the test users into context 2 against those
  groups, and keep (or add) one user in context 1 so isolation still has two
  sides. Decide whether context 1 keeps a full set or only what isolation
  needs.
- **Go integration tests** — `internal/repository` (~69 `security_context`
  references, most of them the literal `1` passed as the scoped argument) and
  `internal/handler` (~40, including the JWTs minted by `signedRequest` /
  `meRequest` and the mock expectations that assert the context reaching the
  repository).
- **pytest** (~35) — the `"securityContext": 1` literals in payloads are
  ignored by the API and should simply go, not flip; what changes is the
  assertions on what comes back, and `conftest.py`'s clients if the seeded
  test users move.
- **Playwright** — the e2e fixture user and `global-setup.ts`, if that user
  moves to context 2.
- **`src/web-app`** (2) — check whether either is a real assumption.
- **`tests/test_security_context_isolation.py`** — keep it, but it must now
  read as "the test tenant (2) cannot see the real tenant (1)", which is the
  right way round and worth stating in the test name.

## Acceptance

- Every suite authenticates as a user in context 2 and asserts 2.
- One isolation test still proves 1 and 2 cannot see each other.
- No literal `1` remains as a *tenant* in a test (column names and ids are
  fine); a grep for `securityContext.*1` in `tests/` finds only the isolation
  case.
- `bin/300-run-tests all` green.

## Deploy needs

None — fixtures and tests only. The scratch databases are rebuilt from the
seed on every run; no migration, no change to any real database.
