# Creating a New Entity Type

This document outlines the steps and key points for creating a new entity type in the ProtoStack application. It covers SQL schema, API server (Go), and web app (TypeScript/React) changes. Reference the `example` entity for working implementations.

## General Notes
- Follow naming conventions: snake_case for DB/JSON, camelCase for TS/Go fields.
- Use optimistic locking with `update_ts`.
- Soft deletes: Set `delete_ts` instead of hard delete.
- Grants: see "Grant id convention" below. Use CRUD verbs: Read, Create, Update, Delete.
- Separate list and edit pages for better UX and navigation.
 - **CRITICAL**: Always use query string parameters with entity-specific names (e.g., `?appUserId=123`, `?appUserGroupId=456`, `?exampleId=789`) for edit and delete operations instead of path parameters (e.g., `/entity/123`) to maintain URL flexibility and avoid changing route patterns. This ensures consistency and prevents routing issues. Do not use generic names like `?entityId=`; always use the specific entity name followed by `Id`.
- **Configuration**: Do not supply default values in config parsing code. All required configuration values must be explicitly set in the environment file to ensure correct setup and easier troubleshooting. Defaults can mask configuration issues.
- Test CRUD operations thoroughly.
- **UI Feedback**: Add success and error toast notifications for all CRUD operations in the respective page components (not in stores). Use the app's notification system (e.g., Mantine notifications) to provide user feedback for create, update, and delete operations.

## Naming Conventions
- **Methods**: Use "Retrieve" for list retrieval (e.g., `RetrieveUserList`) and single-entity retrieval (e.g., `RetrieveUser`). Use Create/Update/Delete for mutations.
- **Stores**: Singular naming (e.g., `useUserStore`, not `useUsersStore`).
- **Pages**: Descriptive names (e.g., `app-user-list-page.tsx`, `app-user-group-edit-page.tsx`).
- **Routes**: `/entityList` for lists, `/entityEdit` for single operations.
- **Grants**: Short codes with CRUD verbs (e.g., "READ_USER_GROUP", "CREATE_USER_GROUP").

## SQL
1. **Create Table**: Use singular table name (e.g., `example`).
   - Standard fields: `<table>_id` (primary key), `security_context_id`, `create_ts`, `create_user_id`, `update_ts`, `update_user_id`, `delete_ts`, `delete_user_id`.
   - Entity-specific fields in snake_case.
   - Example: See `src/api-srv/sql/migrations/0001_initial_schema.sql` for the `example` table; a new entity is a new `src/api-srv/sql/migrations/NNNN_<name>.sql`.
   - **Column types with a worked example** on `example`
     (`migrations/0002_example_jsonb_uuid.sql`):
     - `uuid` — `example_uuid UUID NOT NULL` + unique index. The server
       assigns it (`model.NewUUID()`, crypto/rand v4, no library) in the
       create handler; it is never in an UPDATE's SET list and
       `UpdateExample` `RETURNING`s it so the response carries it. Go type
       `string`; pgx maps uuid <-> string. Web: read-only field, list
       column.
     - `jsonb` — `example_jsonb JSONB` nullable, any JSON value. Go type
       `json.RawMessage` (`omitempty`); pass through
       `repository.jsonbParam` so an absent value is SQL NULL, not the
       JSON literal `null`. Web: a JSON textarea, validated with
       `JSON.parse` in the form, `JSON.stringify(v, null, 2)` on load.
     - arrays — `INTEGER[]` / `FLOAT[]` <-> `[]int` / `[]float64`
       directly; web edits them as comma-separated text.

2. **Grants**: Add schema grants in migration SQL.
   - Everything lives in `public`; grants to `db_reader`/`db_writer` are in `src/api-srv/sql/roles.sql` and the migrations.

3. **Data**: Insert initial data if needed (`src/api-srv/sql/fixtures/local-seed.sql` for a dev database in tenant 1; `test-seed.sql` for the test databases, tenant 2, never 1). The tenant lock covers a new table on its own. Updates to existing data (e.g., adding new grants to user groups) are migrations too: `src/api-srv/sql/migrations/NNNN_<name>.sql`.

## API-SRV
1. **Model** (`src/api-srv/internal/model/<entity>_model.go`):
   - Struct with JSON and DB tags (camelCase JSON, snake_case DB).
   - Example: `ExampleModel` in `example_model.go`.

2. **Repository** (`internal/repository/<entity>_repository.go`):
   - Methods: `RetrieveEntityList`, `CreateEntity`, `UpdateEntity`, `DeleteEntity`.
   - Call pgx directly (`r.db.QueryRow`, `Exec`, `Query`); there is no
     retry wrapper.
   - Errors: return the sentinels in `errors.go`, bare or wrapped with
     `%w` (`ErrNotFound`, `ErrDeleted`, `ErrConflict`, `ErrDuplicate` for
     a unique violation via `isUniqueViolation`, `ErrInUse`,
     `ErrValidation`); use `clientMessage(sentinel, ...)` only when the
     caller needs the detail. Never a bare `fmt.Errorf` a handler must
     read; wrap anything else with `%w` and it answers 500.
   - Optimistic locking: Check `update_ts` on update/delete; on zero rows,
     check the row (same tenant predicate) and return `ErrNotFound`,
     `ErrDeleted` or `ErrConflict`.
   - Example: `ExampleRepository` in `example_repository.go`.

3. **Handler** (`internal/handler/<entity>_handler.go`):
   - Struct with `appsrv`, `entityRepo`.
   - Methods: `RetrieveEntityList`, `CreateEntity`, `UpdateEntity`, `DeleteEntity`.
   - Security: JWT, rate limit, CSRF, grants.
   - Repository errors: `mapRepoError(app, w, err, failMsg, failCode)`;
     no per-handler `errors.Is` switch, never match error text.
   - JSON encoding/decoding.
   - Example: `ExampleHandler` in `example_handler.go`.

4. **Grants** (`src/api-srv/internal/model/grants.go`):
   - Add constants per the "Grant id convention" below: `GrantEntityRead = N`, `GrantEntityCreate = N+1`, `GrantEntityUpdate = N+2`, `GrantEntityDelete = N+3`, N being a multiple of ten.
   - Update `AllKnownGrants` and `GrantIDToShortCodeMap`.
   - Run `bin/305-update-grants-fixture` and add the same ids to
     `src/web-app/constants/grants.ts`; `src/web-app/tests/contract.test.ts`
     fails until both sides agree.
   - Use in handlers.

5. **Routes** (`cmd/server/main.go`):
   - `GET /entityList` → `RetrieveEntityList`
   - `POST /entity` → `CreateEntity`
   - `PUT /entity` → `UpdateEntity`
   - `DELETE /entity` → `DeleteEntity`

6. **Quirks**:
   - Separate handlers for related entities (e.g., `UserGroupHandler` separate from `UserHandler`).
   - Grant ids: see "Grant id convention" below.
   - Delete checks for dependencies (e.g., users in groups).

## WEB-APP

The web half is the *Scaffolding* section of `src/web-app/README.md`: the
files to copy from `example-*` (four plus the store test), the
constants/route/nav entries to add by hand, and the in-page house pattern
(hook order, `HOOK LINE` banner, load by id once per route id, no form for
a record that did not load, list state in the URL, absolute `ROUTES`).
Its *Entity Stores* section holds the store rules (body keys are the Go
`json` tags). Keep it there; do not duplicate it here.

Quirks that still apply:
- Edit pages take their id as a query string with an entity-specific name
  (`?appUserId=`, `?appUserGroupId=`), never a path param; `=new` creates.
- Store naming is singular (`useAppUserStore`); related entities get
  separate stores.
- Confirm deletes with `confirmDelete(entityType, onConfirm)`
  (`utils/confirm-delete.tsx`, `@mantine/modals`); notify success/error.
- Navigation is separate pages via `navigate()`, not conditional rendering.

## Cache Management Pattern

Entity stores built on `stores/create-entity-store.ts` get a `maxUpdateTs`
field for conditional fetching (one name in every store):

- **State**: `maxUpdateTs: number` (initialize to 0)
- **Update Rule**: Only update on non-ID-specific list fetches (full refreshes, not updates, creates or deletes)
- **Usage**: Send as `afterUpdateTs` param for conditional API calls
- **Incremental Cache Updates**: Merge returned data but **don't update max** (preserves detection of other changes)
- **API Support**: Backend filters `WHERE update_ts > afterUpdateTs`

This enables efficient caching while detecting when full refreshes are needed.

## One column, every layer

Adding a column touches each layer once. The worked case is
`app_user.preference_jsonb` (a user's own UI settings, edited through
`PUT /me`); copy the shape, not the names.

| Layer | File | What changed |
|-------|------|--------------|
| Migration | `src/api-srv/sql/migrations/0003_app_user_preference_jsonb.sql` | `ALTER TABLE app_user ADD COLUMN preference_jsonb JSONB` (nullable, no `DEFAULT`) |
| Schema reference | `doc/003-schema-reference.sql` | regenerated by `bin/510-reset-test-db`; commit it with the migration |
| Model | `src/api-srv/internal/model/app_user_model.go` | `PreferenceJSONB json.RawMessage` on `AppUserModel` (`json:"-"`) and on `AppUserOwnProfileModel` (`json:"preferenceJsonb,omitempty"`) |
| Repository | `src/api-srv/internal/repository/user_repository.go` | read in `RetrieveUserByUserName`, left out of the list read; written in `UpdateOwnProfile` through `jsonbParam` |
| Handler | `src/api-srv/internal/handler/profile_handler.go` | request field; `isPreferenceDocument` rejects anything but a JSON object of at most 8 KiB |
| Routes | `src/api-srv/cmd/server/main.go` | none new: `GET /me`, `PUT /me` already existed |
| API doc | `doc/105-api-srv-tech-spec.md` | `PUT /me` row and body |
| Web type | `src/web-app/types/profile-type.ts` | `preferenceJsonb` on `Profile` / `ProfileUpdate`; `resolvePreference` applies defaults |
| Web store | `src/web-app/stores/profile-store.ts` | unchanged: carries the field as received |
| Web hook / page | `src/web-app/hooks/use-preference.ts`, `src/web-app/pages/user-preferences-page.tsx` | read through `resolvePreference`, saved with `updateProfile` |
| Go tests | `profile_handler_test.go`, `user_repository_test.go`, `app_user_model_test.go` | validation, round trip, NULL when unset, absent from the admin shape |
| pytest | `tests/test_own_profile.py` | save, reject a non-object, absent from `/userList` |
| Web tests | `src/web-app/tests/profile-type.test.ts`, `src/web-app/tests/e2e/profile-page.spec.ts` | defaults; save from the page |

## Example References
- **SQL**: `src/api-srv/sql/migrations/0001_initial_schema.sql`, `src/api-srv/sql/fixtures/local-seed.sql`, `src/api-srv/sql/fixtures/test-seed.sql`
- **API-SRV**: `src/api-srv/internal/model/example_model.go`, `src/api-srv/internal/repository/example_repository.go`, `src/api-srv/internal/handler/example_handler.go`
- **WEB-APP**: `types/example-type.ts`, `stores/examples-store.ts`, `pages/example-list-page.tsx`, `pages/example-edit-page.tsx`

Use this as a checklist for new entities. Update as needed for future quirks.

## Grant id convention (new ids only)

A new entity takes the next free block of ten: N is a multiple of ten
that no shipped grant uses, and the ids are

| Offset | Grant  |
|--------|--------|
| N+0    | Read   |
| N+1    | Create |
| N+2    | Update |
| N+3    | Delete |
| N+4..N+9 | extras for that entity (block, reset, ...), in the order added |

Application-wide grants stay outside entity blocks. User and security
entities use the 9xxx range; ordinary entities use 1xxx upward.

Shipped ids are never renumbered. They are stored in group rows, pinned by
`src/api-srv/internal/model/testdata/grants.json` (B054) and mirrored in
`src/web-app/constants/grants.ts`, so a renumber is a data migration. The
shipped ids that break the convention stay as they are: `Example` runs
1001-1004 (block 1000, offset by one), `User` runs CREATE 9001, READ 9002,
UPDATE 9003, DELETE 9004 (older order), 9008, 9009 and 9020 are extras
with unexplained gaps, and 8001 is the own-password grant.
