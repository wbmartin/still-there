---
id: PS0018
title: User profile page: preferences and avatar
status: ready
blocked_by:
waiting_on:
priority: low
domain: web
raised: 2026-09-21
raised_by: PS0006 triage of doc/000-TODO.md
asked:
claimed:
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

## Acceptance

Scope agreed with Brandon and recorded here before work starts.

## Deploy needs

Web build; API + migration only if a preferences table is added.
