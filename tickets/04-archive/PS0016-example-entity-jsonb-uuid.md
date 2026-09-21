---
id: PS0016
title: Add JSONB and UUID columns to the example entity
status: in-review
blocked_by:
waiting_on:
priority: low
domain: api
raised: 2026-09-21
raised_by: PS0006 triage of doc/000-TODO.md
asked:
claimed: 2026-09-21 by TKT-PS0016-example-entity-jsonb-uuid
merged: bdb4c16
deployed: none
validated: 2026-09-21
---
# PS0016 — Example entity should exercise JSONB and UUID

The `example` entity is the template every derived entity is copied from
but had no JSONB or UUID column, so a derived app met those types with
no worked example of model tag, pgx scan, migration or web-side schema.
Both types were added end to end — migration, model, repository, handler,
TypeScript schema, edit page — as a random server-assigned UUID and a
free-form JSONB column, coordinated to land ahead of the PS0010 entity
rewrite.

## What changed
- Migration `0002_example_jsonb_uuid.sql`: `example_uuid UUID`,
  `example_jsonb JSONB`; existing rows backfilled with
  `gen_random_uuid()`; `example_uuid` then `NOT NULL` with a unique index.
- `model.NewUUID()` (`internal/model/uuid.go`): 20 lines of `crypto/rand`
  in canonical form, no dependency.
- `ExampleModel` gains `ExampleUUID`/`ExampleJSONB`; all six repository
  column enumerations extended; `jsonbParam` maps an absent
  `json.RawMessage` to SQL NULL; `UpdateExample` now `RETURNING
  example_uuid`.
- `CreateExample`/`CreateExampleList` assign the uuid server-side; a
  client-supplied value is overwritten and never appears in a SET list.
- Fixtures seed a uuid and a jsonb value; web edit page gets a read-only
  uuid field and a JSON textarea validated with `JSON.parse`; list page
  shows the uuid column.
- Tests: Go repository/model round-trip, pytest `test_jsonb_and_uuid`,
  Playwright `example-pages.spec.ts`; `doc/810` gained a worked-example
  section for uuid/jsonb/array column types.

## Rules that survived
- A server-assigned identity column is overwritten after decode and never
  placed in an UPDATE's SET list — a client value is always ignored, on
  create and update alike.
- `json.RawMessage(nil)` marshals to the JSON literal `null`, not SQL
  NULL; a dedicated `jsonbParam` helper is required wherever an optional
  jsonb column is written, to keep absent and `null` distinguishable.
- `gen_random_uuid()` is core Postgres since 13 — prefer it over an
  extension or a Go uuid library for a server-generated id.

## Figures at close
| | date |
|---|---:|
| go tests | 165 (2026-09-21) |
| unit (vitest) | 28 (2026-09-21) |
| e2e (Playwright) | 6 (2026-09-21) |
| pytest | 64 (2026-09-21) |

## Endpoints
- `POST/PUT/GET /example`, `POST/PUT/GET /exampleList` — JWT + example
  grants, unchanged; responses/bodies gain `exampleUuid` (response only,
  input ignored) and `exampleJsonb` (optional, any JSON value).

## Security
No new endpoint, auth path or dependency. The uuid is proven
non-guessable and non-writable by the client (crypto/rand, 122 random
bits, never in a SET list); jsonb input relies on the existing body-size
limit and Postgres's own parser, with no schema by design since the
column is a free-form worked example; every query still scopes by
`security_context_id`. `example` remains demo data — a derived app
copying the pattern onto a real entity must decide separately whether
its jsonb column may carry PII.

## Left open
- The local dev database (`loc_proto_db`) was never stamped by the
  migrator; migration `0002` needs a hand-applied `status`/`up` with the
  DDL role before the next `bin/600-bundle-uat`.
- `doc/810`'s content is still due the PS0010 rewrite of the entity
  template; this ticket only fixed the worked example within it.

## Sources
- `tickets/04-archive/engineering-log/260921_1750-TKT-PS0016-example-entity-jsonb-uuid.md`
- git: `bdb4c16`
