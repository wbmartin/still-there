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

## Requirements

Both types end to end on `example`, as the worked pattern a derived entity
copies. Brandon 2026-09-21: random uuid, server-assigned.

## What changed and why

- **Migration `0002_example_jsonb_uuid.sql`:** `example_uuid UUID` and
  `example_jsonb JSONB` added; existing rows backfilled with
  `gen_random_uuid()` (PG13+ built-in, no extension); `example_uuid` then
  `NOT NULL` with a unique index. No `DEFAULT` — the app sets it (doc/900).
- **`model.NewUUID()`** (`internal/model/uuid.go`): crypto/rand v4 in
  canonical form; 20 lines instead of a dependency. Lives in `model` so
  repository tests can use it without an import cycle.
- **`ExampleModel`:** `ExampleUUID string` (`exampleUuid`),
  `ExampleJSONB json.RawMessage` (`exampleJsonb,omitempty`) — the JSON
  value is emitted inline in responses and accepted verbatim in bodies.
- **Repository:** all six column enumerations extended. `jsonbParam`
  turns an absent `RawMessage` into SQL NULL (json.Marshal of a nil
  RawMessage would store the literal `null`). `example_uuid` is never in
  a SET list; `UpdateExample` now `RETURNING example_uuid` (QueryRow, not
  Exec; `pgx.ErrNoRows` takes the old rows-affected==0 path) so the
  update response carries the row's uuid rather than whatever the client
  sent.
- **Handler:** `CreateExample` / `CreateExampleList` assign the uuid; a
  client-supplied value is overwritten. Updates need nothing — the
  column is not writable.
- **Fixtures:** seed rows carry `gen_random_uuid()` and one jsonb value.
- **Web:** `Example.exampleUuid?`, `exampleJsonb?: unknown`; the edit
  page has a monospace JSON textarea validated with `JSON.parse` in the
  form (`Must be valid JSON (or empty)`), pretty-printed on load, parsed
  on submit, empty → absent; a read-only UUID field on existing rows;
  the list page shows a UUID column.
- **Tests:** repository round-trip (create/read, update keeps uuid and
  rewrites jsonb, nil → NULL → nil, duplicate uuid rejected); model JSON
  shape and `NewUUID` format/uniqueness; pytest `test_jsonb_and_uuid`
  (client uuid ignored on create and update, non-object JSON, clear);
  Playwright `example-pages.spec.ts` (create with JSON → server uuid in
  the list → edit round-trip → invalid JSON refused → delete). Every
  existing repo-test constructor now sets `ExampleUUID`.
- **`doc/810`:** SQL section gained *Column types with a worked example*
  (uuid, jsonb, arrays) pointing at these files.

## Files modified

`src/api-srv/sql/migrations/0002_example_jsonb_uuid.sql` (new),
`sql/fixtures/local-seed.sql`, `internal/model/{example_model,uuid}.go`,
`internal/model/example_model_test.go`,
`internal/repository/example_repository{,_test}.go`,
`internal/handler/example_handler.go`, `src/web-app/types/example-type.ts`,
`pages/example-{edit,list}-page.tsx`, `tests/e2e/example-pages.spec.ts`
(new), `tests/test_a_example_crud.py`, `doc/810-create-new-entity-type-notes.md`.

## Endpoints introduced or modified

No new endpoints. `POST/PUT/GET /example`, `POST/PUT/GET /exampleList`
(JWT + example grants, unchanged) gain two body/response fields:
`exampleUuid` (response only; ignored on input) and `exampleJsonb`
(any JSON value, optional). Data classification unchanged — example rows
are demo data; `exampleJsonb` is whatever the caller stores, scoped by
security context like every other column.

## Checks

`go build`, `go vet`, `tsc`, `eslint` clean; `bin/300-run-tests all`:
go=165 unit=28 e2e=6 py=64.

## Acceptance

- `example` has a `jsonb` and a `uuid` column, round-tripped by the Go
  and Playwright suites.
- `doc/810` (or its PS0010 replacement) shows the pattern.

## Deploy needs

Go binary, web build, migration `0002`.

**Hand step before `bin/600-bundle-uat`:** the presales deploy ships a
dump of the local `loc_proto_db`, so 0002 must be applied there first,
as the DDL role (the app role cannot create `schema_migration`):

```
cd src/api-srv
go run ./cmd/migrate -url postgres://postgres:<pw>@localhost:5432/loc_proto_db status
go run ./cmd/migrate -url ... stamp 1      # only if 0001 shows as pending: the schema pre-dates the migrator
go run ./cmd/migrate -url ... up
```

`bin/510-reset-test-db` already proves the chain on an empty database.
