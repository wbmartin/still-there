# PS0016 — Security Considerations

Branch `TKT-PS0016-example-entity-jsonb-uuid`. Profiles: **2. Authenticated
API endpoint** (existing example endpoints gain two fields), **6. Data
layer change** (migration, new columns, six queries), **1. UI feature**.
No new endpoint, dependency, or config.

## 2. Authenticated API endpoint

- **Auth unchanged:** every example handler still runs `CheckJWT` → rate
  limit → `CheckCSRF` → `CheckAuthorization(GrantExample*)` →
  `CheckAllChecksComplete`. Nothing was added outside that chain.
- **Client cannot choose the uuid:** `CreateExample` / `CreateExampleList`
  overwrite `ExampleUUID` with `model.NewUUID()` after decode; no UPDATE
  statement lists `example_uuid`. pytest sends a uuid on both create and
  update and asserts it was ignored. The uuid is therefore not a
  capability token or a guessable id — it is an opaque server identity,
  unique by index.
- **Randomness:** `crypto/rand`, 122 random bits, version/variant bits
  set per RFC 4122. `crypto/rand` failure panics rather than returning a
  weak value; that is the correct failure mode for an exhausted entropy
  source and matches what `uuid.New()` in google/uuid does.
- **Input validation on jsonb:** the body decoder already rejects
  malformed JSON (400); `exampleJsonb` is any valid JSON value or absent.
  There is no schema on the value by design (it is the template's
  worked example of a free-form column). Size is bounded by the existing
  request-body limit; Postgres `jsonb` parses and normalises it, so a
  pathological document costs the API nothing beyond the body read.
- **No over-fetch:** responses add exactly the two columns.
- **Scoping:** every SELECT/UPDATE still filters `security_context_id`;
  the unique index on `example_uuid` is global, which is fine — the
  value is random, not tenant-derived, and never used as a lookup key.

## 6. Data layer change

- **Parameterised:** all six statements use `$n` placeholders; the jsonb
  bytes go in as a parameter (`jsonbParam`), never interpolated.
- **Migration safety:** `ADD COLUMN` (nullable) → backfill → `SET NOT
  NULL` → `CREATE UNIQUE INDEX`, one transaction via the migrator.
  Table is tiny in every environment; on a large table the index build
  would be the lock to think about. Forward only; no data loss.
- **NULL vs `null`:** `jsonbParam` maps an absent value to SQL NULL so
  "no value" and "the JSON literal null" stay distinguishable; the repo
  test proves nil → NULL → nil.
- **Classification:** `example` is demo data. A derived app copying
  this pattern onto a real entity should decide then whether the jsonb
  column may carry PII; nothing here changes the answer for `example`.

## 1. UI feature

- **Rendering:** the JSON value is shown in a `<Textarea>` as text and
  the uuid in a read-only `<TextInput>` / table cell — React escapes
  both; no HTML rendering of the stored document.
- **Client-side parse:** `JSON.parse` on user text is the form
  validator; a failure is a validation message, not an exception path.
  The parsed value is sent as JSON, not evaluated.
- **Grants:** the page's `useAuthGuard` grants are unchanged; the new
  fields sit inside the existing create/update forms.

## What did I miss?

- `exampleUuid` appears in list responses for anyone with `READ_EXAMPLE`
  — it reveals nothing (random) and is the intended public identity.
- Bulk update (`PUT /exampleList`) returns a message, not rows, so it
  never echoes a client uuid.
- The migration's `gen_random_uuid()` is core Postgres since 13; UAT is
  17. No extension, no new role privilege.
