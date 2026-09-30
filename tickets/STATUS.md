# Ticket status

Where every ticket stands, generated from the ticket headers. *Questions for
the customer* is for the customer; the engineering sections are for whoever
picks up the next piece of work. Do not edit below the marker.

<!-- BEGIN GENERATED -- scripts/lint-tickets.py --update-status -->

*Generated 2026-09-30 18:28 EDT by `scripts/lint-tickets.py --update-status`.*

## Questions for the customer — 0 open

## Engineering — ready to start (2)

| Ticket | Title | Priority | Domain |
|---|---|---|---|
| **PS0022** | Move every automated test into tenant 2 (security_context_id 2 is the reserved test tenant) | medium | data |
| **PS0021** | MFA loose ends — web grant 9020 has no Go grant, resetUserMFA leaves mfa_setup true, successful login never resets the lockout counters | low | security |

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
| data | 1 |

**Next free number:** `PS0023` (engineering), `PS1001` (Ask Customer).

<!-- END GENERATED -->
