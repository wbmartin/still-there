---
id: PS0017
title: DEFERRED — Passkey (WebAuthn) as an additional second factor
status: blocked
blocked_by:
waiting_on: a customer request, then TOTP e2e coverage
priority: low
domain: security
raised: 2026-09-21
raised_by: PS0006 triage of doc/000-TODO.md
asked:
claimed:
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

## Constraint (Brandon, 2026-09-21)

The passkey is **in addition to TOTP, never a replacement.** A user keeps
their TOTP enrolment. Two readings were sized:

- **(a) AND** — TOTP *and* passkey both required at the MFA step.
- **(b) OR** — TOTP always enrolled; a passkey is an optional alternative
  that satisfies the MFA step. Recovery for a lost passkey is TOTP.

## Investigation (2026-09-21)

### The archived plan is not a second-factor design

`251221_0607-passkey-plan.md` is a **passwordless primary login** ("Sign in
with Passkey" button, l.408/422; rollout step 5 "make passkeys the default
authentication method", l.482). It never mentions the temp-token step.
Under the constraint above its `/webauthn/authenticate/*` primary path is
dropped outright. What survives: begin/finish endpoint shape, self-service
list/delete, `go-webauthn/webauthn` + `@simplewebauthn/browser`, a
preferences-page management UI, an enable flag.

Stale against today's code: `REFERENCES app_user`, `DEFAULT`s, `created_at`
(all against doc/900 — table would be `app_user_passkey` with
`security_context_id`, `create_ts` set in Go, PK `app_user_passkey_id`);
migration numbering (`0003_…` embedded in `cmd/migrate`); `app_lib`
package (now `app.Services`, handler-struct style per PS0008/09); "encrypt
public keys" (unnecessary); library versions; challenge storage
unspecified (in-process KV keyed by temp token is the natural place —
same single-instance assumption as the temp token).

### Current flow the plan missed

`POST /auth` → for non-exempt users a 32-byte `tempToken` in the KV for
10 min, no JWT (`auth_handler.go:254-303`) → `POST /mfaSetup` /
`POST /mfaVerify` accept `tempToken` (SkipJWT) or a JWT; verify success on
the temp path mints the JWT + cookie + CSRF and expires the token
(`user_mfa_handler.go:593-620`). `mfa_exempt` skips the step entirely;
`/admin/resetUserMFA` wipes only `totp_seed` (not `mfa_setup`); lockout is
`loginlock-ip-*`/`loginlock-user-*` in the KV. Web: `authState` gains
`"MFA"` + `tempToken`; `login.tsx` → `/mfa-setup`;
`mfa-onboarding-page.tsx` does both enrol and verify in one page and sets
`authState:"YES"` directly.

### What it would take

Shared: migration `0003_app_user_passkey.sql` (`credential_id BYTEA
UNIQUE`, `public_key`, `sign_count`, `transports`, `aaguid`,
`backup_eligible/state`, `device_name`, `last_used_ts` + house columns);
config `PASSKEY_ENABLED`, `PASSKEY_RP_ID`, `PASSKEY_RP_ORIGIN_LIST` in
every `etc/*-api-srv.env` + doc/905; endpoints `POST /passkeyRegisterBegin
/Finish` (JWT+CSRF, in-session after TOTP), `POST /passkeyVerifyBegin
/Finish` (temp-token path mirroring `/mfaVerify`: rate limit, lockout,
`incrementFailedAttempt`, JWT on success), `GET /passkeyList`,
`DELETE /passkey`; `/auth` temp response gains `passkeyEnrolled`;
`/admin/resetUserMFA` must also soft-delete passkeys. Go: `auth/webauthn.go`,
model, repository, handler, main wiring, config (~8 files + migration),
one Go dep with a heavier transitive set (cbor, go-tpm) — a deliberate
add, consistent with `otp`/`jwt` for protocol work. Web: `@simplewebauthn/
browser`, api constants, auth types/store, MFA page branch, preferences
page + list/register components (~8 files). Tests: Go handler tests with
mocked repo; pytest can only gate the begin endpoints; Playwright can run
the full ceremony via CDP `WebAuthn.addVirtualAuthenticator` (Chromium
only) — **but only once a non-exempt e2e fixture user with a known TOTP
seed exists; today `tests/conftest.py:42-47` bans non-exempt users and
`test_auth_flow.py:110-112` is a stub. TOTP has no automated coverage.**

- **(b) OR — size M**, ~16–20 files. `/mfaVerify` untouched; MFA page
  offers "Use passkey" when enrolled, falls back to the code.
- **(a) AND — size L**, ~20–24 files. `/mfaVerify` must stop minting the
  JWT and hand back a kept/new temp token for the passkey step; forced
  enrolment at first login; lost passkey = admin reset only.

Risks either way: RP ID/origin must match the browser-visible host per
environment (a wrong UAT value silently orphans every passkey); sign-count
regression handling and synced-passkey counter 0; exempt users must be
refused enrolment as `/mfaSetup` does; e2e is Chromium-only.

### Prior art

No customer has asked. The only recorded answer is "TOTP"
(`251204_0723-react-web-app-idea.md:74-75`); no PS1xxx question exists.
`251130_2039-api-server-code-review.md:65-83` flags the global
`TOTP_ENCRYPTION_KEY` as High — passkeys (public keys only) sidestep it,
a mild argument for (b).

## Recommendation

**Defer.** The plan is unusable as-is; the real design is (b), a solid
M-size API + web pair. Nothing technical blocks it, but: no customer has
asked and this is a template — the feature belongs in a derived app
unless a customer names it; and the TOTP path it would branch from has
zero automated coverage. Lift the deferral when: (1) a customer asks
(file a `PS1xxx-ASK-CUSTOMER` if plausible), (2) a non-exempt e2e fixture
user + TOTP Playwright coverage exist (worth its own small ticket now),
(3) reading (b) is agreed and UAT/prod hostnames for the RP ID are known.
The archived plan's passwordless scope is dropped explicitly.

## Decision

**Deferred (Brandon, 2026-09-21).** Back in `01-open/` as `blocked`; not
dropped. Pick it up again as reading (b) when the three conditions above
are met. The investigation stands; do not redo it — re-check
`user_mfa_handler.go` and the `"MFA"` web step for drift first.

## Acceptance

Decide build / defer / drop and record it here; if build, split into
the API and web tickets it needs.

## Deploy needs

None for the investigation.
