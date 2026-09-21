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
merged:
deployed:
validated:
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

## Acceptance

Scope agreed with Brandon and recorded here before work starts.

## Deploy needs

Go binary, web build, migration `0003`.
