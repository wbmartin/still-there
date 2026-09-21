---
id: PS0001
title: Backport infra, ticketing and tooling from derivative apps
status: ready
blocked_by:
waiting_on:
priority: high
domain: infrastructure
raised: 2026-09-21
raised_by: Brandon
asked:
claimed: 2026-09-21 by TKT-PS0001-backport-derivatives
merged:
deployed:
validated:
---
# PS0001 — Backport infra, ticketing and tooling from derivative apps

Branch: `TKT-PS0001-backport-derivatives` → `main`

## Description

Placeholder — the `pr-ticket-docs` skill will flesh this section out
(What / Root cause or requirements / What changed and why / Files
modified / Endpoints introduced or modified / Deploy needs) once the work
on this branch is complete.

## What

ProtoStack is the generic starter template that other apps (eastern-rebate,
atticus-meeting, gifttaggle, ...) are derived from. Several of those
derivatives have since built infrastructure and process that belongs back
in the template, generalized. This ticket backports it:

- **Migrate binary + DDL/CI role model** (from atticus-meeting) — schema
  changes move to `src/api-srv/cmd/migrate` with numbered
  `src/api-srv/sql/migrations/NNNN_name.sql` files embedded in the binary,
  plus `roles.sql` and `fixtures/`, replacing hand-run DDL.
- **Ticketing system** (from eastern-rebate) — `tickets/`, `scripts/lint-tickets.py`,
  and the `ticket-workflow` / `ticket-triage` / `compact-tickets` / `handoff`
  skills, genericized: `ER` → `PS`, "Eastern" → "the customer", named
  contacts → generic recipient roles.
- **Presales bundle/deploy scripts** (from eastern-rebate) — `bin/600-bundle-uat`
  and `bin/610-deploy-uat` for packaging and shipping a build to the UAT
  server.
- **Generic Go/React/test-infra fixes** (from atticus-meeting) — assorted
  fixes to the API server, web app, and test tooling that are not
  app-specific.

## Files modified

See the branch diff; this section will be completed by `pr-ticket-docs`.

## Endpoints

None expected from the ticketing/docs portion of this work. Any API
changes from the migrate-binary or generic-fixes portions are documented
here when they land.

## Deploy needs

TBD — depends on which pieces land on this branch. At minimum: no schema
change from the ticketing docs themselves; the migrate-binary work, if
merged here, needs a Go binary rebuild.
