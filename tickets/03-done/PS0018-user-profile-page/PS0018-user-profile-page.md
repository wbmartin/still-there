---
id: PS0018
title: User profile page: preferences and avatar
status: in-review
blocked_by:
waiting_on:
priority: low
domain: web
raised: 2026-09-21
raised_by: PS0006 triage of doc/000-TODO.md
asked:
claimed: 2026-09-21 by TKT-PS0018-user-profile-page
merged: 8cb03c7
deployed: none
validated: 2026-09-21
---
# PS0018 — User profile page

**Source:** `doc/000-TODO.md` ("user profile page — preferences, upload
avatar button"), triaged 2026-09-21 by PS0006. Brandon: file it.

## What

`pages/user-preferences-page.tsx` exists; the TODO wanted a profile page
with preferences and an avatar upload. Avatar upload needs somewhere to
put files — the template has no file storage, and the "nginx serves
non-sensitive files" idea was dropped at the same triage — so scope this
to preferences first and treat the avatar as a follow-on that carries its
own storage decision.

## Scope (agreed with Brandon, 2026-09-21)

1. **Self-service profile:** `GET /me` and `PUT /me` — JWT + CSRF +
   `LOGON`, the caller's own row only (id from claims, never from the
   body), editable `firstName`, `lastName`, `email`, optimistic lock on
   `updateTs`. No migration.
2. **Stored preferences:** `app_user.preference_jsonb` (migration
   `0003`, the PS0016 jsonb pattern), carried on `/me`; first keys
   `colorScheme` (`light` | `dark` | `auto`) applied through Mantine, and
   `rowsPerPage` used by the three list pages.
3. **Page:** `user-preferences-page` becomes the profile page — sections
   Profile / Preferences / Security / Administration — same route.
4. **Out:** avatar upload — needs a file-storage decision; follow-on
   ticket when a derived app has one.

## Requirements

Scope above. A user edits their own name/email and two UI preferences;
nothing admin-only becomes reachable; the preference applies app-wide
on the next load without visiting the page.

## What changed and why

- **Migration `0003_app_user_preference_jsonb.sql`:** `app_user.
  preference_jsonb JSONB`, NULL until saved. No grant change (table-level
  grants in 0001 cover new columns).
- **`AppUserModel.PreferenceJSONB json.RawMessage`** (`preferenceJsonb,
  omitempty`); both `app_user` SELECTs and scans carry it.
- **Repository:** `RetrieveUserByID(ctx, userID, securityContext)` (the
  read behind `/me`; `ErrNotFound` on miss) and `UpdateOwnProfile(...)`
  — a narrow UPDATE of `first_name, last_name, email, preference_jsonb`
  with optimistic lock (`ErrConflict` / `ErrNotFound` sentinels, unlike
  the older narrow updates), `update_user_id` = the user themselves,
  `jsonbParam` for NULL-vs-`null`.
- **`ProfileHandler`** (`profile_handler.go`, own small
  `ProfileRepositoryInterface`): `GET /me`, `PUT /me`. Full explicit
  chain, `GrantLogon` only. The row is always `claims.UserID` /
  `claims.SecurityContext` — an `appUserId` in the body is ignored.
  Validation: names required, ≤200 chars, email via `mail.ParseAddress`
  in plain `user@host` form only, `updateTs` required, `preferenceJsonb`
  absent or a JSON object ≤ 8 KiB; body capped at 12 KiB. Responses
  marshal `AppUserModel`, so `pwHash`/`totpSeed` stay `json:"-"`.
  Routes sit with the other own-scoped endpoints in `main.go`; doc/105
  table updated.
- **Web:** `API_ENDPOINTS.ME`; `types/profile-type.ts` (`Profile`,
  `Preference{colorScheme, rowsPerPage}`, `resolvePreference` with
  defaults auto/15); `stores/profile-store.ts` (zustand, registered
  cache clearer); `hooks/use-preference.ts` (`usePreference`,
  `useRowsPerPage`). `app.tsx` loads the profile when `authState` becomes
  `YES` (and on context switch) and applies `colorScheme` through
  `useMantineColorScheme`; `main.tsx` `defaultColorScheme="auto"`. The
  three list pages take `itemsPerPage` from `useRowsPerPage()`.
- **`user-preferences-page.tsx`** (same route, nav label "Profile"):
  Profile form (first/last/email) + Preferences (colour scheme select,
  rows per page 5–200) with one Save that PUTs `/me` and resets the
  form from the stored row; then Security (context switch, change
  password), Administration, Session (logout, clear cache).
- **`index.css`:** the Vite-template `:root`/`a`/`button` colour rules
  and the `prefers-color-scheme: light` block are gone — they hard-coded
  a palette that would have fought a forced light/dark scheme; Mantine's
  `styles.css` owns body colours now.
- **Tests:** Go handler (check sequence, LOGON-only, id from claims,
  secrets absent, six 400 cases, 409, trimmed fields reach the repo),
  repo round-trip (NULL → doc → lock conflict → wrong context), pytest
  `test_own_profile.py` as the basic user (body `appUserId` ignored,
  409, three 400s, restore + clear), Playwright `profile-page.spec.ts`
  (save → dark applies on `/dashboard` after navigation → values persist
  → restore).

## Files modified

`src/api-srv/sql/migrations/0003_app_user_preference_jsonb.sql` (new),
`internal/model/app_user_model.go`, `internal/repository/user_repository
{,_test}.go`, `internal/handler/profile_handler{,_test}.go` (new),
`cmd/server/main.go`, `doc/105-api-srv-tech-spec.md`,
`tests/test_own_profile.py` (new), `src/web-app/{app,main}.tsx`,
`index.css`, `constants/api.ts`, `types/profile-type.ts` (new),
`stores/profile-store.ts` (new), `hooks/use-preference.ts` (new),
`pages/user-preferences-page.tsx`, `pages/*-list-page.tsx` (3),
`tests/e2e/profile-page.spec.ts` (new).

## Endpoints introduced or modified

| Method | Path | Auth | CSRF | Grant | Data |
|---|---|---|---|---|---|
| GET | `/me` | JWT | safe method | LOGON | caller's own row: name, email, groups, MFA flags, preferences — PII of the caller only |
| PUT | `/me` | JWT | yes | LOGON | writes first/last/email/preferenceJsonb of the caller's row |

Introduced locally; reaches UAT with the next deploy. `GET /userList`
also now carries `preferenceJsonb` for admins with `READ_USER`.

## Checks

`go build`, `go vet`, `tsc`, `eslint` clean; `bin/300-run-tests all`:
go=180 unit=28 e2e=7 py=66.

## Acceptance

Scope agreed with Brandon and recorded here before work starts.

## Deploy needs

Go binary, web build, migration `0003`. Same local hand step as PS0016:
apply with the migrator on `loc_proto_db` as the DDL role before the
next `bin/600-bundle-uat` (`status` → `up`).
