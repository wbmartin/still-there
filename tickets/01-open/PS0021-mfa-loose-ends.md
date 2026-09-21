---
id: PS0021
title: MFA loose ends — web grant 9020 has no Go grant, resetUserMFA leaves mfa_setup true, successful login never resets the lockout counters
status: needs-triage
blocked_by:
waiting_on:
priority: low
domain: security
raised: 2026-09-21
raised_by: PS0017 investigation
asked:
claimed:
merged:
deployed:
validated:
---
# PS0021 — Three small MFA inconsistencies

**Source:** PS0017 investigation, 2026-09-21. Filed for Brandon to
triage; each is a one-line fix but two change behaviour.

## Found

1. **Grant 9020 exists only on the web.** `constants/grants.ts:21`
   `RESET_MFA_TOKEN: 9020` gates the Reset-MFA button
   (`app-user-edit-page.tsx:313`); `grants.go` has no 9020, and
   `POST /admin/resetUserMFA` checks `GrantUserUpdate` (9003). The seed
   gives Admin 9020 (`local-seed.sql`), so it works by fixture. Either
   add `GrantResetUserMFA = 9020` in Go and make the handler check it
   (a real separate permission), or gate the button on `UPDATE_USER`
   and drop 9020 from the web and the seed. Recommended: the former —
   resetting someone's second factor deserves its own grant.
2. **`/admin/resetUserMFA` clears `totp_seed` but not `mfa_setup`**
   (`user_mfa_handler.go:727-733`). Works today only because
   `/mfaSetup` regenerates on an empty seed and the temp-token verify
   path ignores `mfa_setup`. Set `mfa_setup = false` in the same UPDATE
   (`UpdateUserTOTPSeed` already does when called with a seed — check
   which path the reset uses).
3. **Successful login never resets the failed-attempt counters:**
   `auth_handler.go:246-251` has the `resetFailedAttempts` call
   commented out, so failures accumulate for the whole
   `FAILED_LOGIN_ATTEMPT_WINDOW_DURATION` regardless of successes in
   between. PS0011 lists the function as dead code; the fix is to call
   it, not delete it. Decide the intended semantics (reset on success is
   the usual one) and wire it.

## Acceptance

Brandon picks a resolution for each; PS0020's tests cover 2 and 3.

## Deploy needs

Go binary; web build for item 1; no migration (the grant table is code).
