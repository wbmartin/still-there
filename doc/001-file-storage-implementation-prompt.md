# OpenCode CLI Prompt: File Download Handler with Dev/Prod Toggle (Go + Nginx)

# This has not been reviewe d by a human yet.

You are generating production-quality Go code for a **file download handler** in a service built with the standard `net/http` package (no frameworks).

The system follows this architecture:

- **Go service = control plane**
  - Authenticates the request
  - Authorizes access to a file via database lookup
  - Sets HTTP headers (Content-Type, Content-Disposition, Cache-Control)
  - Either:
    - streams the file directly from disk (DEV mode), or
    - delegates file serving to Nginx using `X-Accel-Redirect` (UAT/PROD)

- **Nginx = data plane (UAT/PROD only)**
  - Serves bytes from disk
  - Uses an `internal` location for private files

The code must be **readable, explicit, and boring**. Prefer clarity over cleverness.

---

## Requirements

### 1. Environment Toggle
Implement a configuration option:

- Environment variable: `FILE_SERVE_MODE`
  - `"direct"` → stream file directly from disk (DEV)
  - `"accel"` → set `X-Accel-Redirect` header (UAT/PROD)

Default to `"accel"` if the variable is unset.

Only the *final file-serving step* should differ between modes.
All auth, lookup, and header logic must be shared.

---

### 2. HTTP Endpoint

Implement:
GET /api/files/{object_id}

Where:
- `{object_id}` is a GUID (opaque identifier)
- The handler must:
  1. Authenticate the request (stub is fine)
  2. Authorize the user for the object (stub is fine)
  3. Look up file metadata from Postgres
  4. Set appropriate response headers
  5. Serve the file according to `FILE_SERVE_MODE`

---

### 3. File Storage Model

- Files are stored on disk using **GUID filenames**
- Disk path is derived from DB metadata, not user input
- Human-readable filenames are stored in the database and used only for
  `Content-Disposition`

Example:
- Disk: `/data/private/ab/cd/<guid>`
- Header: `Content-Disposition: inline; filename="Quarterly Report.pdf"`

---

### 4. HTTP Header Behavior

Set the following headers consistently in both modes:

- `Content-Type` (from DB, fallback to `application/octet-stream`)
- `Content-Disposition`
  - default to `inline`
  - include human-readable filename
- `Cache-Control`
  - use `private, no-store` for private files

Do **not** write the response body in accel mode.

---

### 5. Security Requirements

- Prevent path traversal (`..`)
- Do not trust client-supplied filenames or content-types
- Do not expose real disk paths to clients
- In accel mode:
  - Use `X-Accel-Redirect`
  - Redirect only to a fixed internal prefix (e.g. `/_internal/private/`)

---

### 6. Code Structure

Generate:

- A small config loader for `FILE_SERVE_MODE`
- A `FileMeta` struct
- Stubbed `authorize()` and `lookupFileMeta()` functions
- A `downloadFileHandler(db *sql.DB)` returning `http.HandlerFunc`
- A minimal `main()` wiring with `http.ServeMux`

No frameworks. Use only:
- `net/http`
- `database/sql`
- standard library packages

---

## Postgres Schema (INCLUDE THIS SECTION)

Also generate a **SQL section** (commented or separate markdown section) describing
the minimal Postgres tables required.

At minimum, include:

### `objects` table
Fields should include:
- `id` (UUID / text GUID, primary key)
- `storage_relpath` (relative disk path, e.g. `ab/cd/<guid>`)
- `content_type`
- `original_filename`
- `size_bytes`
- `created_at`

### Optional Notes
- Explain why GUID filenames are used on disk
- Explain why the human filename lives in the DB
- Mention that ACLs can be added later via a join table

Provide SQL `CREATE TABLE` statements compatible with Postgres.

---

## Output Expectations

- Output **only** Go source code and SQL (no commentary)
- Code should compile (stubs allowed)
- Keep the handler logic linear and easy to read
- Prefer explicit error handling over helpers
- Assume this is a foundation for a long-lived codebase

Generate now.
