# Ticket status

Where every ticket stands, generated from the ticket headers. *Questions for
the customer* is for the customer; the engineering sections are for whoever
picks up the next piece of work. Do not edit below the marker.

<!-- BEGIN GENERATED -- scripts/lint-tickets.py --update-status -->

*Generated 2026-09-21 15:04 EDT by `scripts/lint-tickets.py --update-status`.*

## Questions for the customer — 0 open

## Engineering — ready to start (8)

| Ticket | Title | Priority | Domain |
|---|---|---|---|
| **PS0003** | Sunset the shrmpl kv, log and vault clients and all their local tooling | high | tooling |
| **PS0005** | One agent rulebook — fold doc/9xx guidance into CLAUDE.md, fix or retire opencode.json | high | docs |
| **PS0006** | Move ProtoStack history out of doc/ into the engineering log; fix every broken path | high | docs |
| **PS0009** | Explicit per-handler security checks are by design — record it; fix the exceptions' loose ends; make ExampleHandler mockable | medium | api |
| **PS0012** | Web — fix the entity-page bugs; document that pages are deliberate scaffolding | medium | web |
| **PS0013** | Web hygiene — unused deps, dead schemas/types, console noise, generated grants/endpoints, naming, modal API | medium | web |
| **PS0014** | Test harnesses, bin/ numbering and env-file contract, tmp/ conventions, coverage gaps | medium | tooling |
| **PS0015** | sql/ and cmd/ shape — role files, fixture naming, migrate `new`, load-test out of cmd/ | medium | api |

## Engineering — merged, awaiting validation (2)

| Ticket | Title | Merged | Deployed |
|---|---|---|---|
| **PS0002** | Security and correctness hygiene found in the 2026-09-21 template review | ea5f4f0 | — |
| **PS0004** | Document the presales → post-sales deploy migration; fix bin/600 fail-fast | ef91f35 | — |

## Engineering — validated, awaiting archive (1)

| Ticket | Title | Validated |
|---|---|---|
| **PS0001** | Backport infra, ticketing and tooling from derivative apps | 2026-09-21 |

## Engineering — blocked (4)

| Ticket | Title | Priority | Waiting on |
|---|---|---|---|
| **PS0007** | Single rename point and a "deriving a new app" checklist | high | PS0003, PS0004 |
| **PS0008** | Split internal/app_lib into packages named by role (no KV/log interfaces) | medium | PS0003 |
| **PS0010** | Make the example entity a lean, copyable template — error mapping, scoped queries, retry removal, doc/810 rewrite | low | Brandon |
| **PS0011** | API dead code, duplicated helpers, table-driven config loader, lint baseline | low | Brandon |

## Open tickets by domain

| Domain | Open |
|---|---|
| api | 5 |
| docs | 4 |
| tooling | 2 |
| web | 2 |
| security | 1 |

**Next free number:** `PS0016` (engineering), `PS1001` (Ask Customer).

<!-- END GENERATED -->
