# PS0002 — Security Considerations

Branch `TKT-PS0002-security-correctness-hygiene`. Profiles applied: **2**
(authenticated endpoints), **3/4** (anonymous rate limiting, lockout), **1**
(UI), **8** (config / secrets).

## 2. Authenticated endpoints

**`GET /userList` no longer returns `totpSeed`.** Response is narrowed, never
widened. The seed was AES-GCM ciphertext, so exposure needed the
`TOTP_ENCRYPTION_KEY` too — which item 7 shows *was* in git alongside it.
Treat every seed as compromised for anyone who had repo read access;
rotation of the key (below) forces MFA re-enrolment anyway. No client
decodes `totpSeed` (grep clean in `src/web-app`); the seed only moves via
`UpdateUserTOTPSeed`. `TestRetrieveUserList_OmitsSecrets` asserts absence.

**`DELETE /userGroup`** now returns the real id. Caller already knew it
(they sent it); no new information.

## 3/4. Anonymous rate limiting and lockout

`CheckRateLimitAndUserBlock` with nil claims now keys `RPM*<ip>*<minute>`
instead of `RPM*anon*<minute>`. Effects:

- **Fail closed unchanged.** JWT failure still blocks before this runs;
  KV errors still fail open as before (`RATE_LIMIT_FAIL`), unchanged.
- **Global-bucket DoS removed.** One hostile client could previously 429
  the MFA step for every user at 20 req/min. Now they exhaust only their
  own IP.
- **Per-IP bucket is weaker than global against a distributed attacker** —
  20 req/min × N IPs. Same posture as `CheckRateLimit` already had; login
  lockout (`isAccountLocked`, per-username as well as per-IP) is the
  control that matters for credential stuffing, and it is untouched.
- **`admin` exemption kept** — deliberate (Brandon 2026-09-21).

### ⚠ X-Forwarded-For is client-controlled — pre-existing, now wider

`ClientIP` takes the **first** XFF entry. Both nginx configs
(`infra/uat/etc/nginx_uat-proto.example.com:44`, `infra/prd/…:44`) use
`$proxy_add_x_forwarded_for`, which **appends** the real peer IP to
whatever the client sent. So `X-Forwarded-For: 10.0.0.1` from the client
arrives as `10.0.0.1, <real>` and we key on `10.0.0.1`. A client can pick
its own bucket per request and evade IP-keyed limits and the per-IP login
lockout entirely (per-username lockout still holds).

This was already true of `handler.getClientIP` (login lockout, MFA verify)
and `CheckRateLimit` (2 handlers). This ticket does not make it *worse* for
those paths and does not fix it; it does extend the same spoofable key to
the 27 `CheckRateLimitAndUserBlock` handlers, whose previous global bucket
was trivially DoS-able but not evadable.

**Recommended follow-up (one line, decision for Brandon):** behind this
nginx, the trustworthy entry is the **last** one. Either take
`parts[len(parts)-1]`, or have nginx overwrite rather than append
(`proxy_set_header X-Forwarded-For $remote_addr`). Direct exposure without
a proxy (local dev) should ignore XFF altogether. Not done here because it
changes lockout semantics beyond the agreed scope.

## 1. UI

`generateStrongPassword` now uses `crypto.getRandomValues` with rejection
sampling (bytes ≥ 210 discarded so the 70-char alphabet has no modulo
bias). 12 chars × log2(70) ≈ 73 bits. No new client-side secrets; no
rendering change.

`use-auth-guard`: the TBD branch now correctly skips the token refresh on
`/login`. Grant enforcement (`YES` branch) is unchanged; the backend
`CheckAuthorization` is the real control and is untouched.

## 8. Config / secrets

**Live secrets were tracked:** `etc/vault-configs/dev_proto-…` held a
`DATABASE_URL` with password, `JWT_SECRET`, and `TOTP_ENCRYPTION_KEY`.
Untracked here, `.example` with placeholders committed, folder gitignored
(`*.example` excepted). `etc/loc-shrmpl-vault-srv.env` still points
`CONFIG_DIR` at the folder, so local dev keeps working off the on-disk copy.

**They remain in git history** (every clone, GitHub). Untracking does not
revoke them. Required, outside this PR:

1. Rotate `JWT_SECRET` (invalidates all sessions — users re-login).
2. Rotate `TOTP_ENCRYPTION_KEY` (existing seeds become undecryptable —
   users re-enrol MFA, or re-encrypt under the new key first).
3. Change the `DATABASE_URL` password.
4. Optionally purge from history (`git filter-repo`) and force-push; only
   worth it if the repo has ever been shared beyond Brandon.

These were `dev_` values; if UAT/prd reuse any of them the rotation is
urgent rather than hygienic.

`tests/reports/*` HTML contained no secrets (checked: request logs, no
tokens). Root `package.json` removal has no security effect.

## What did I miss?

- `ClientIP` returns `RemoteAddr` verbatim when `SplitHostPort` fails; the
  key is built with `fmt.Sprintf`, so a weird value can only make a weird
  KV key, not an injection — KV commands are space-delimited but the key
  is one token and IPs have no spaces; a hostile XFF with a space fails
  `net.ParseIP` and falls through to `RemoteAddr`.
- Removing the debug `Printf` also stopped leaking `SecurityContext` ids to
  stdout/journal. Minor.
- IPv6 clients behind NAT/CGNAT share buckets — accepted, same as before.
