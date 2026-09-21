# Ticket status

Where every ticket stands, generated from the ticket headers. *Questions for
the customer* is for the customer; the engineering sections are for whoever
picks up the next piece of work. Do not edit below the marker.

<!-- BEGIN GENERATED -- scripts/lint-tickets.py --update-status -->

*Generated 2026-09-21 12:40 EDT by `scripts/lint-tickets.py --update-status`.*

## Questions for the customer — 0 open

## Engineering — ready to start (12)

| Ticket | Title | Priority | Domain |
|---|---|---|---|
| **PS0002** | Security and correctness hygiene found in the 2026-09-21 template review | high | security |
| **PS0003** | Decide shrmpl's place in the template and quarantine or remove it | high | tooling |
| **PS0004** | One deploy story — make presales 600/610 the degenerate case of the infra/ pipeline | high | deploy |
| **PS0005** | One agent rulebook — fold doc/9xx guidance into CLAUDE.md, fix or retire opencode.json | high | docs |
| **PS0006** | Split doc/ into template content vs ProtoStack history; fix every broken path | high | docs |
| **PS0009** | Replace the AppHelper check state machine with Protected(grant) middleware; one handler style; route registry | high | api |
| **PS0010** | Make the example entity a lean, copyable template — error mapping, scoped queries, retry removal, doc/810 rewrite | high | api |
| **PS0012** | Web — shared entity list/edit scaffolding, route registry, route-level auth guard, feature folders | high | web |
| **PS0008** | Split internal/app_lib into packages named by role | medium | api |
| **PS0011** | API dead code, duplicated helpers, table-driven config loader, lint baseline | medium | api |
| **PS0013** | Web hygiene — unused deps, dead schemas/types, console noise, generated grants/endpoints, naming, modal API | medium | web |
| **PS0014** | Test harnesses, bin/ numbering and env-file contract, tmp/ conventions, coverage gaps | medium | tooling |

## Engineering — validated, awaiting archive (1)

| Ticket | Title | Validated |
|---|---|---|
| **PS0001** | Backport infra, ticketing and tooling from derivative apps | 2026-09-21 |

## Engineering — blocked (2)

| Ticket | Title | Priority | Waiting on |
|---|---|---|---|
| **PS0007** | Single rename point and a "deriving a new app" checklist | high | PS0003, PS0004 |
| **PS0015** | sql/ and cmd/ shape — role files, fixture naming, migrate `new`, load-test out of cmd/ | low | PS0004 |

## Open tickets by domain

| Domain | Open |
|---|---|
| api | 5 |
| docs | 3 |
| tooling | 2 |
| web | 2 |
| security | 1 |
| deploy | 1 |

**Next free number:** `PS0016` (engineering), `PS1001` (Ask Customer).

<!-- END GENERATED -->
