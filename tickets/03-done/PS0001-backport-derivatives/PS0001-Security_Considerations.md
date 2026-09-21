# PS0001 — Security Considerations

**Profiles applied:** 8 Infrastructure/Config, 6 Data Layer, 4 Auth & Permission Logic, 2 Authenticated API Endpoint, 3 Public Endpoint, 1 UI Feature, 7 Third-Party Dependency.

**Verdict:** Net security improvement. The migrator separates DDL from the runtime role, pre-auth request bodies are now bounded, a CSRF-token log leak is closed, and the web app no longer spams an anonymous endpoint. Residuals are all confined to local-dev/presales conveniences and are listed below.

## 8. Infrastructure / Config

- **Secrets in committed files.** `etc/loc-test.env` (docker `loc-postgres` superuser password), `etc/e2e-api-srv.env` and `src/api-srv/sql/roles.sql` (`proto_app_svc_account` password) contain well-known local-only values. The svc-account password was already committed in the old `sql/010-initialcreate.sql`; nothing new is exposed. `etc/e2e-api-srv.env` binds `127.0.0.1` and comments that it must never run outside a laptop. `etc/uat-api-srv.env.example` carries `SERVERONLY` placeholders only; the convention (repo file is a redacted mirror of the host file) is documented in the header and `doc/320`.
- **Network exposure.** `SERVER_BIND_ADDRESS` lets a deployed API bind loopback behind nginx; default (empty) is unchanged. Playwright's API instance binds 127.0.0.1:8082 with CORS limited to `:5182`.
- **Presales deploy (`610-deploy-uat`) runs as root** by design: installs a systemd unit (`User=www-data`, `NoNewPrivileges`, `ProtectSystem=strict`, `ReadWritePaths` = log dir only), drops and recreates the app DB from the bundle's `db.sql`, replaces binaries under `libexec/` (root:root 0755). It refuses to run without an existing `etc/api-srv.env` and never overwrites it. Rollback copies (`.prev`) stay on host. The bundle travels over `scp` (ssh); transport security is the operator's ssh key.
- **Bundle contents.** `600-bundle-uat` dumps the *local* database — whatever data is on the developer laptop ships to UAT. Presales only; the skill and `doc/320` say the webhook/registry + migrator path is the post-agreement upgrade. `--no-db` exists for when UAT holds data that must be kept.
- **`infra/uat`, `infra/prd` placeholders.** 35 files; every credential/hostname/registry/webhook value is `SERVERONLY` or `example.com` (grep-verified). Scripts derive `SRV_ROOT` from their own path, source `etc/*.env`, and die on any missing key — no `${VAR:-default}` so a half-configured host cannot silently deploy with a guess. Restart is via a root-owned systemd `.path` unit watching `stg/restart.request`, so the deploy user needs no sudo. Not exercised on any host yet (placeholders); must be reviewed again when a derived app fills them.
- **Optional log receiver.** With `LOG_RECEIVER_HOST_PORT` unset, logs go to stdout/journald only — no silent loss, but no central log shipping either; hosts that want it must set the key.

## 6. Data Layer

- **Migrator role model.** `proto_ddl` owns the schema and is the only role that runs `migrate up`; the API's `proto_app_svc_account` keeps `db_reader`/`db_writer` (DML only, cannot ALTER/DROP). `ddl-role-host-setup.sql` moves ownership per object and sets `ALTER DEFAULT PRIVILEGES` so future tables inherit grants. `migrate -config` reads only `DATABASE_URL` (`ReadEnvFile`) so the DDL credential file never needs `JWT_SECRET` etc.
- **Migration safety.** Each file runs in one transaction with its `schema_migration` row; SHA-256 checksums fail `verify`/`up` on an edited applied file; an advisory lock serialises concurrent deploys; out-of-order versions are refused. `stamp` records without executing — operator-only, documented in `doc/050`/`900`.
- **Grant ordering fix.** In the old `010` file, `GRANT ... ON ALL TABLES` ran before any table existed and was a no-op; `0001_initial_schema.sql` grants after creation, so a fresh DB now actually has reader/writer grants (verified: `proto_app_svc_account` SELECT/INSERT succeed on a `510` scratch DB).
- **Queries.** `scoped_query.go` builds parameterised `WHERE security_context_id = $1 AND delete_user_id IS NULL [AND id IN ($2..)]` — no string interpolation of user input; generated SQL identical to before. `ErrDuplicate` mapping reads `pgconn.PgError.Code` only; no error text reaches the client beyond a fixed `DUPLICATE_ERROR`.
- **Scratch DB guards.** `510/513/514` refuse names not ending `_test_db`/`_e2e_db`; `514` TRUNCATEs only inside those.

## 4. Auth & Permission Logic

- **Session refresh.** `authenticateWithAuthToken` clears auth only on 401/403; network failure or 5xx sets `UNREACHABLE` and keeps the existing (possibly valid) session client-side. Fails closed on the server: every request still passes JWT/CSRF checks; the client merely stops *voluntarily* discarding its cookie on transient errors. Backward compatible with existing tokens.
- **Session-expired listener.** Any mutation receiving 401/403 triggers a local sign-out (CSRF cleared, caches cleared, state reset) and a notification — removes the "stuck signed-in UI" state. Re-entrancy guarded.
- **CSRF logging.** `CheckCSRF` no longer logs the expected token (derived from session id) — closes an info leak into log files/receivers.
- **Body caps.** Login and security-context switch decode JSON before lockout/rate-limit checks; now bounded at 4 KiB. Tested against authorised and unauthorised personas by the pytest suite (63) and Playwright (wrong-password path).

## 2 / 3. Endpoints (authenticated / public)

- No new endpoints. `POST /error-report` (anonymous) now capped at 8 KiB and no longer called by the shipped web app; it remains registered and rate-limited as before. Worst case unchanged: log noise, now bounded.
- 409 on duplicate user/group create/update replaces a 500. Error body reveals only that a duplicate exists within the caller's own security context (group names are scoped per context; usernames are global — the same information the UI already exposes via the list endpoints).

## 1. UI Feature

- `ErrorBoundary` renders `error.message` only, never the stack, and no longer posts client stack traces to an unauthenticated endpoint. The `UNREACHABLE` alert renders no server data.
- No new client-side secrets; `VITE_API_PROXY_URL` is a dev-only proxy target.

## 7. Third-Party Dependency

- `@playwright/test` ^1.63 added as a **devDependency** only; downloads Chromium at test time. Not shipped. No new Go modules.

## Residuals accepted

1. Local-only Postgres passwords remain in `etc/loc-test.env`, `etc/e2e-api-srv.env`, `roles.sql` (pre-existing pattern; loopback-only usage; every derived app must rotate on real hosts — `600-bundle-uat` reads the password from the local env rather than committing a host value).
2. `610-deploy-uat` is a root-run, DB-replacing script; acceptable for presales UAT with no customer data, not for post-agreement environments (doc'd).
3. `proto_ddl`/`proto_ci` creation is a manual runbook step (`ddl-role-host-setup.sql`), not automated — deliberate, mirrors atticus-meeting.

## What did I miss?

- Checked whether `ReadEnvFile` could be pointed at an arbitrary file to exfiltrate — it only runs in `cmd/migrate` with an operator-supplied path; no network surface.
- Checked that `updateItem` throwing (instead of swallowing) doesn't leak server error text into toasts: pages show `readErrorMessage` output which is the API's already-sanitised `message` field.
- Checked `600-bundle-uat` MANIFEST: records branch/commit/tool versions and DB name only — no credentials.
- `.gitignore` now ignores `**/tests/e2e/.auth/` so Playwright's saved session cookie never gets committed.
