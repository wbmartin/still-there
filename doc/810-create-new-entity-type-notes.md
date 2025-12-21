# Creating a New Entity Type

This document outlines the steps and key points for creating a new entity type in the ProtoStack application. It covers SQL schema, API server (Go), and web app (TypeScript/React) changes. Reference the `example` entity for working implementations.

## General Notes
- Follow naming conventions: snake_case for DB/JSON, camelCase for TS/Go fields.
- Use optimistic locking with `update_ts`.
- Soft deletes: Set `delete_ts` instead of hard delete.
- Grants: Create entity-specific grants in grants.go starting at appropriate numbers (e.g., 1001+ for typical new entities, 9000+ for user/security groups. each new entity should start with ###0 for READ, ###1 for CREATE, ###2 UPDATE, ###3 for DELETE so that the 10s place groups them together with room in between for additional grants). Use CRUD verbs: Read, Create, Update, Delete.
- Separate list and edit pages for better UX and navigation.
 - **CRITICAL**: Always use query string parameters with entity-specific names (e.g., `?appUserId=123`, `?appUserGroupId=456`, `?exampleId=789`) for edit and delete operations instead of path parameters (e.g., `/entity/123`) to maintain URL flexibility and avoid changing route patterns. This ensures consistency and prevents routing issues. Do not use generic names like `?entityId=`; always use the specific entity name followed by `Id`.
- Test CRUD operations thoroughly.

## Naming Conventions
- **Methods**: Use "Read" for list retrieval (e.g., `ReadUserList`) and single-entity retrieval (e.g., `ReadUser`). Use Create/Update/Delete for mutations.
- **Stores**: Singular naming (e.g., `useUserStore`, not `useUsersStore`).
- **Pages**: Descriptive names (e.g., `app-user-list-page.tsx`, `user-group-edit-page.tsx`).
- **Routes**: `/entityList` for lists, `/entityEdit` for single operations.
- **Grants**: Short codes with CRUD verbs (e.g., "READ_USER_GROUP", "CREATE_USER_GROUP").

## SQL
1. **Create Table**: Use singular table name (e.g., `example`).
   - Standard fields: `entity_id` (primary key), `security_context`, `create_ts`, `create_user_id`, `update_ts`, `update_user_id`, `delete_ts`, `delete_user_id`.
   - Entity-specific fields in snake_case.
   - Example: See `sql/010-initialcreate.sql` for `example` table.

2. **Grants**: Add schema grants in migration SQL.
   - `GRANT USAGE ON SCHEMA biz TO group_global_writer;`
   - `GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA biz TO group_global_writer;`
   - `GRANT ALL ON ALL SEQUENCES IN SCHEMA biz TO group_global_writer;`
   - Repeat for `appcore` schema.

3. **Data**: Insert initial data if needed (e.g., `sql/015-initialdata.sql`). For updates to existing data (e.g., adding new grants to user groups), create update SQL files in `sql/` (e.g., `sql/update_admin_grants.sql`).

## API-SRV
1. **Model** (`internal/model/entity_model.go`):
   - Struct with JSON and DB tags (camelCase JSON, snake_case DB).
   - Example: `ExampleModel` in `example_model.go`.

2. **Repository** (`internal/repository/entity_repository.go`):
   - Methods: `ReadEntityList`, `CreateEntity`, `UpdateEntity`, `DeleteEntity`.
   - Use `ExecuteWithRetry` for queries.
   - Optimistic locking: Check `update_ts` on update/delete.
   - Example: `ExampleRepository` in `example_repository.go`.

3. **Handler** (`internal/handler/entity_handler.go`):
   - Struct with `appsrv`, `entityRepo`.
   - Methods: `ReadEntityList`, `CreateEntity`, `UpdateEntity`, `DeleteEntity`.
   - Security: JWT, rate limit, CSRF, grants.
   - JSON encoding/decoding.
   - Example: `ExampleHandler` in `example_handler.go`.

4. **Grants** (`internal/model/grants.go`):
   - Add constants: `GrantEntityRead = 1xx0`, `GrantEntityCreate = 1xx1`, `GrantEntityUpdate = 1xx2`, `GrantEntityDelete = 1xx3`.
   - Update `AllKnownGrants` and `GrantIDToShortCodeMap`.
   - Use in handlers.

5. **Routes** (`cmd/server/main.go`):
   - `GET /entityList` → `ReadEntityList`
   - `POST /entity` → `CreateEntity`
   - `PUT /entity` → `UpdateEntity`
   - `DELETE /entity` → `DeleteEntity`

6. **Quirks**:
   - Separate handlers for related entities (e.g., `UserGroupHandler` separate from `UserHandler`).
   - Grants start at 1001 for examples, 9010 for user and security related entities..
   - Delete checks for dependencies (e.g., users in groups).

## WEB-APP
1. **Constants**:
    - **Routes** (`constants/routes.ts`): Define route paths without leading `/` (e.g., `'appUserList'`).
    - **API Endpoints** (`constants/api.ts`): Define API paths (e.g., `'/userList'`).

3. **Types** (`types/entity-type.ts`, one file per entity):
    - `Entity` interface (camelCase).
    - `ApiEntityModel` for API responses.
    - Example: `example-type.ts`.

 4. **Store** (`stores/entity-store.ts`):
     - Zustand store with `entity[]`, `maxEntityUpdateTs`, `loading`, `error`.
     - Functions: `createEntity`,`readEntityList`,  `updateEntity`, `deleteEntity`.
     - API calls to `/entityList`, `/entityEdit`.
     - Cache management: `maxEntityUpdateTs` updated only on full list fetches. Only set `maxEntityUpdateTs` in list retrieval functions. CRUD operations should update local cache without modifying the timestamp to avoid cache invalidation issues.
     - Example: `user-store.ts`.

5. **Pages**:
   - **List**: `entity-list-page.tsx` - Table with pagination, add/edit/delete buttons.
   - **Edit**: `entity-edit-page.tsx` - Form for create/edit, with save/cancel/delete.
   - Use query params for IDs, `navigate` for transitions.
   - Example: `app-user-list-page.tsx`, `app-user-edit-page.tsx`.

6. **Routing** (`App.tsx`):
    - `/entityList` → `EntityListPage`
    - `/entityEdit?entityId=#` → `EntityEditPage` (use Query string parameters with entity-specific names like ?appUserId=, ?appUserGroupId=, etc., DO NOT USE PATH PARAMS)
   - Update nav links.
   - 

7. **Components**:
    - Reuse `DeleteConfirmModal` with `entityType`.
    - Notifications for success/error.

8. **Quirks**:
   - Store naming: Singular (e.g., `useUserStore`, not `useUsersStore`).
   - Separate stores for related entities (e.g., `useUserGroupsStore`).
   - Function comments: `//============================^FUNCTION LINE ^============================`
   - Edit forms: Multi-select for arrays (e.g., groups), password fields for new users.
   - Navigation: Separate pages with `navigate()` instead of conditional rendering.
   - Grants centralized in `grants.go`, models define their own maps if needed.

## Cache Management Pattern

All entity stores must include a `max<Entity>UpdateTs` field for conditional fetching:

- **State**: `max<Entity>UpdateTs: number` (initialize to 0)
- **Update Rule**: Only update on non-ID-specific list fetches (full refreshes, not updates, creates or deletes)
- **Usage**: Send as `afterUpdateTs` param for conditional API calls
- **Incremental Cache Updates**: Merge returned data but **don't update max** (preserves detection of other changes)
- **API Support**: Backend filters `WHERE update_ts > afterUpdateTs`

This enables efficient caching while detecting when full refreshes are needed.

## Example References
- **SQL**: `sql/010-initialcreate.sql`, `sql/015-initialdata.sql`
- **API-SRV**: `internal/model/example_model.go`, `internal/repository/example_repository.go`, `internal/handler/example_handler.go`
- **WEB-APP**: `types/example-type.ts`, `stores/user-store.ts`, `pages/app-user-list-page.tsx`, `pages/app-user-edit-page.tsx`

Use this as a checklist for new entities. Update as needed for future quirks.
