# PS0018 — Security Considerations

Branch `TKT-PS0018-user-profile-page`. Profiles: **2. Authenticated API
endpoint** (two new: `GET /me`, `PUT /me`), **6. Data layer change**
(migration 0003, new column, narrow update), **1. UI feature**. No new
dependency or config.

## 2. Authenticated API endpoint

- **Enforcement:** both handlers spell the full chain — `CheckJWT` →
  `CheckRateLimitAndUserBlock` → `CheckCSRF` (GET is a safe method and
  passes by design; PUT needs the header) → `CheckAuthorization(GrantLogon)`
  → `CheckAllChecksComplete`. Proven by `TestProfileHandler_CheckSequence`
  (401 without cookie, 403 PUT without CSRF, 403 without LOGON, repository
  never reached until all pass).
- **Authorization scope — the central point:** the row is selected by
  `claims.UserID` and `claims.SecurityContext`, both from the signed JWT.
  The body's `appUserId` is not read (pytest sends `appUserId: 1` as the
  basic user and gets their own row back). A user cannot read or write
  another user's row through `/me`, in any context. `LOGON` is the only
  grant because every signed-in user is entitled to their own profile;
  admin-band grants (`READ_USER`, `UPDATE_USER`) stay on `/userList` and
  `PUT /user`.
- **Privilege boundary of the write:** `UpdateOwnProfile` SETs exactly
  `first_name, last_name, email, preference_jsonb` (+ audit columns). A
  user cannot change `user_name`, group memberships, `mfa_exempt`,
  `mfa_setup`, `totp_seed`, `pw_hash` or `security_context_id` through
  it — those need the admin endpoint or the password/MFA flows. The
  repo test asserts user name and groups are untouched after the update.
- **Input validation:** names non-empty and ≤200 (column width); email
  through `net/mail.ParseAddress` and required to equal the parsed
  address (rejects `Name <a@b>`), ≤200; `updateTs` required (no blind
  overwrite); `preferenceJsonb` absent or a JSON **object** of ≤8 KiB
  (a list/scalar is refused, so the column stays a document the UI can
  read by key); request body capped at 12 KiB with `MaxBytesReader`.
- **Response scope:** `AppUserModel` is marshalled directly, so `pwHash`
  and `totpSeed` remain `json:"-"`; the handler test asserts both are
  absent. The row includes the caller's own groups and MFA flags — data
  they already see about themselves.
- **Logging:** the helper logs the outcome (`ME_RD_*`, `ME_UP_*`); auth
  failures are logged by the checks as everywhere else.
- **Email is not verified.** Changing it takes effect immediately with
  no confirmation mail — the template has no mail path. Nothing in the
  app keys on email (login is by user name), so the blast radius is a
  wrong contact address. A derived app that uses email for recovery must
  add verification before exposing `PUT /me`.

## 6. Data layer change

- **Parameterised:** all statements use `$n`; the jsonb document is a
  parameter via `jsonbParam`.
- **Migration:** one nullable `ADD COLUMN`; no backfill, no lock of
  note, forward only. No grant change (table-level grants).
- **Scope of reads:** `RetrieveUserByID` filters `security_context_id`
  and `delete_user_id IS NULL`; a soft-deleted or foreign-context user
  is `ErrNotFound`, tested.
- **Classification:** `preference_jsonb` is UI settings (colour scheme,
  page size). It is free-form up to 8 KiB, so a user *could* store
  arbitrary text there; it is only ever read back to that same user (and
  to admins via `/userList`). Not PII by design; not rendered as HTML.
- **`GET /userList` now includes `preferenceJsonb`** for admins — the
  same audience that already sees the row. Acceptable; noted.

## 1. UI feature

- **Rendering:** name/email in Mantine inputs (escaped), preferences
  through a `Select` limited to three values and a bounded
  `NumberInput`; an unknown `colorScheme` in a stored document resolves
  to `auto` (`isColorScheme` / `resolvePreference`). No HTML rendering of
  stored text.
- **Colour-scheme plumbing:** `setColorScheme` receives only
  `light|dark|auto`; Mantine writes it to `localStorage`
  (`mantine-color-scheme-value`) as a convenience — that is a per-browser
  echo of the server value, not a source of truth, and is overwritten on
  the next profile load.
- **Grants on the page:** unchanged (`LOGON` gate; admin links still
  disabled without `CREATE_USER*`).
- **Cache hygiene:** the profile store registers a cache clearer, so a
  sign-out or context switch drops the previous user's name/email and
  preferences before the next session's load.

## What did I miss?

- Rate limiting is the per-user limiter; a user hammering `PUT /me` is
  throttled like any other authenticated call.
- The profile load in `app.tsx` runs on every `YES` transition, so a
  refreshed tab re-reads it — one small GET, no fan-out.
- `updateUserId` is set to the caller, which is correct audit: the user
  changed their own row.
- Nothing to rotate; no secrets, no config, no dependency.
