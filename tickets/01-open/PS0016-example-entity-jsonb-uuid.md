---
id: PS0016
title: Add JSONB and UUID columns to the example entity
status: ready
blocked_by:
waiting_on:
priority: low
domain: api
raised: 2026-09-21
raised_by: PS0006 triage of doc/000-TODO.md
asked:
claimed:
merged:
deployed:
validated:
---
# PS0016 — Example entity should exercise JSONB and UUID

**Source:** `doc/000-TODO.md` ("you need to add JSONB, UUID to the example
handler"), triaged 2026-09-21 by PS0006. Brandon: file it.

## What

The `example` entity is the template every derived entity is copied from
(PS0010). It has no JSONB and no UUID column, so a derived app meets
those types with no worked example of model tag, pgx scan, migration and
web-side schema.

Add one of each to `example` end to end: migration, `ExampleModel`,
repository scan/insert/update, handler validation, TypeScript schema and
edit page. Coordinate with PS0010 — if the entity template is being
rewritten, do this as part of it rather than before.

## Acceptance

- `example` has a `jsonb` and a `uuid` column, round-tripped by the Go
  and Playwright suites.
- `doc/810` (or its PS0010 replacement) shows the pattern.

## Deploy needs

Go binary, web build, one migration.
