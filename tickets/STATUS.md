# Ticket status

Where every ticket stands, generated from the ticket headers. *Questions for
the customer* is for the customer; the engineering sections are for whoever
picks up the next piece of work. Do not edit below the marker.

<!-- BEGIN GENERATED -- scripts/lint-tickets.py --update-status -->

*Generated 2026-09-21 19:55 EDT by `scripts/lint-tickets.py --update-status`.*

## Questions for the customer — 0 open

## Engineering — ready to start (1)

| Ticket | Title | Priority | Domain |
|---|---|---|---|
| **PS0021** | MFA loose ends — web grant 9020 has no Go grant, resetUserMFA leaves mfa_setup true, successful login never resets the lockout counters | low | security |

## Engineering — validated, awaiting archive (17)

| Ticket | Title | Validated |
|---|---|---|
| **PS0001** | Backport infra, ticketing and tooling from derivative apps | 2026-09-21 |
| **PS0002** | Security and correctness hygiene found in the 2026-09-21 template review | 2026-09-21 |
| **PS0003** | Sunset the shrmpl kv, log and vault clients and all their local tooling | 2026-09-21 |
| **PS0004** | Document the presales → post-sales deploy migration; fix bin/600 fail-fast | 2026-09-21 |
| **PS0005** | One agent rulebook — fold doc/9xx guidance into CLAUDE.md, fix or retire opencode.json | 2026-09-21 |
| **PS0006** | Move ProtoStack history out of doc/ into the engineering log; fix every broken path | 2026-09-21 |
| **PS0007** | Single rename point and a "deriving a new app" checklist | 2026-09-21 |
| **PS0008** | Split internal/app_lib into packages named by role (no KV/log interfaces) | 2026-09-21 |
| **PS0009** | Explicit per-handler security checks are by design — record it; fix the exceptions' loose ends; make ExampleHandler mockable | 2026-09-21 |
| **PS0012** | Web — fix the entity-page bugs; document that pages are deliberate scaffolding | 2026-09-21 |
| **PS0013** | Web hygiene — unused deps, dead schemas/types, console noise, generated grants/endpoints, naming, modal API | 2026-09-21 |
| **PS0014** | Test harnesses, bin/ numbering and env-file contract, tmp/ conventions, coverage gaps | 2026-09-21 |
| **PS0015** | sql/ and cmd/ shape — role files, fixture naming, migrate `new`, load-test out of cmd/ | 2026-09-21 |
| **PS0016** | Add JSONB and UUID columns to the example entity | 2026-09-21 |
| **PS0018** | User profile page: preferences and avatar | 2026-09-21 |
| **PS0019** | ReadEnvFile keeps surrounding quotes; APP_NAME reaches the TOTP issuer with quotes | 2026-09-21 |
| **PS0020** | TOTP MFA has no automated coverage — non-exempt fixture user and a Playwright enrol/verify test | 2026-09-21 |

## Engineering — blocked (3)

| Ticket | Title | Priority | Waiting on |
|---|---|---|---|
| **PS0010** | Make the example entity a lean, copyable template — error mapping, scoped queries, retry removal, doc/810 rewrite | low | Brandon |
| **PS0011** | API dead code, duplicated helpers, table-driven config loader, lint baseline | low | Brandon |
| **PS0017** | DEFERRED — Passkey (WebAuthn) as an additional second factor | low | a customer request, then TOTP e2e coverage |

## Open tickets by domain

| Domain | Open |
|---|---|
| api | 2 |
| security | 2 |

**Next free number:** `PS0022` (engineering), `PS1001` (Ask Customer).

<!-- END GENERATED -->
