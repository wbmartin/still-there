---
id: PS0017
title: Passkey (WebAuthn) as a second factor
status: ready
blocked_by:
waiting_on:
priority: low
domain: security
raised: 2026-09-21
raised_by: PS0006 triage of doc/000-TODO.md
asked:
claimed: 2026-09-21 by TKT-PS0017-passkey-second-factor
merged:
deployed:
validated:
---
# PS0017 — Passkey as a second factor

**Source:** `doc/000-TODO.md` ("investigate passcode as second factor -
doc in 120"), triaged 2026-09-21 by PS0006. Brandon: file it.

## What

TOTP is the only second factor. The design sketch is the archived
`tickets/04-archive/engineering-log/251221_0607-passkey-plan.md` (was
`doc/120-passkey-plan.md`); no passkey code exists. Investigate first —
the plan predates the current MFA flow (`MFASetupHandler`,
`MFAVerifyHandler`, temp-token step) and should be re-read against it.

## Acceptance

Decide build / defer / drop and record it here; if build, split into
the API and web tickets it needs.

## Deploy needs

None for the investigation.
