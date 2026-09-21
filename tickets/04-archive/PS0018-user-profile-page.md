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

The TODO wanted a profile page with preferences and an avatar upload;
avatar upload was scoped out (the template has no file storage) and
deferred to a follow-on ticket once a derived app has a storage
decision. Delivered: self-service `GET/PUT /me` scoped to the caller's
own row, a `preference_jsonb` column carrying colour scheme and rows-per
page, and `user-preferences-page.tsx` turned into the profile page with
Profile/Preferences/Security/Administration sections.

## What changed
- Migration `0003_app_user_preference_jsonb.sql`: `app_user.
  preference_jsonb JSONB`, nullable, no backfill.
- `ProfileHandler` (`GET /me`, `PUT /me`): full explicit check chain,
  `GrantLogon` only, row always `claims.UserID`/`claims.SecurityContext`
  — a body `appUserId` is never read.
- `UpdateOwnProfile` repository method: narrow UPDATE of
  `first_name, last_name, email, preference_jsonb` with optimistic lock
  on `updateTs` (`ErrConflict`/`ErrNotFound` sentinels).
- Validation: names required ≤200 chars, email via
  `net/mail.ParseAddress` in plain form, `preferenceJsonb` absent or a
  JSON object ≤8 KiB, body capped 12 KiB.
- Web: `profile-store.ts` (zustand), `use-preference.ts`
  (`usePreference`/`useRowsPerPage`), `app.tsx` applies `colorScheme`
  through Mantine on every `authState` → `YES` transition; the three
  list pages take `itemsPerPage` from the stored preference.
- `index.css`: removed the Vite-template hard-coded light/dark palette
  and `prefers-color-scheme` override, which would have fought a saved
  scheme — Mantine's `styles.css` now owns body colours.
- Tests: Go handler (check sequence, LOGON-only, id from claims, secrets
  absent, 6× 400, 409), repo round-trip, pytest `test_own_profile.py`,
  Playwright `profile-page.spec.ts` (scheme checked on `/dashboard`
  after navigation, not on the setting page).

## Rules that survived
- A self-service "own row" endpoint reads the row id from the signed JWT
  claims, never from the request body — proven by sending a foreign
  `appUserId` and getting the caller's own row back.
- A narrow self-service UPDATE lists only the columns a user may change;
  admin-only columns (group membership, `mfa_exempt`, secrets) stay
  reachable only through the admin endpoint.
- A jsonb-backed UI preference resolves an unknown/invalid stored value
  to a safe default (`resolvePreference`) rather than failing the page.
- Framework-default CSS that hard-codes a colour scheme must be removed
  before an app-controlled colour-scheme preference is introduced, or
  the two fight each other.

## Figures at close
| | date |
|---|---:|
| go tests | 180 (2026-09-21) |
| unit (vitest) | 28 (2026-09-21) |
| e2e (Playwright) | 7 (2026-09-21) |
| pytest | 66 (2026-09-21) |

## Endpoints
- `GET /me` — JWT, safe method (CSRF not required), `LOGON` — returns
  the caller's own row: name, email, groups, MFA flags, preferences.
- `PUT /me` — JWT, CSRF, `LOGON` — writes the caller's
  `firstName`/`lastName`/`email`/`preferenceJsonb` only.
- `GET /userList` unchanged in auth but now also carries
  `preferenceJsonb` for admins with `READ_USER`.

## Security
Both new endpoints run the full explicit check chain and scope strictly
to `claims.UserID`/`SecurityContext`; a user cannot read or write another
user's row, and the self-service UPDATE cannot touch `user_name`, group
membership, `mfa_exempt`, `totp_seed`, `pw_hash` or
`security_context_id`. Response marshalling keeps `pwHash`/`totpSeed`
tagged `json:"-"`. One residual: email is not verified on change (the
template has no mail path) — acceptable here since login is by user
name and nothing keys on email, but a derived app using email for
recovery must add verification before exposing `PUT /me`.

## Left open
- Avatar upload — needs a file-storage decision; follow-on ticket when a
  derived app has one.

## Sources
- `tickets/04-archive/engineering-log/260921_1905-TKT-PS0018-user-profile-page.md`
- git: `8cb03c7`
