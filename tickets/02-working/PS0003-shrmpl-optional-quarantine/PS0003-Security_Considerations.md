# PS0003 — Security Considerations

Branch `TKT-PS0003-shrmpl-optional-quarantine`. Profiles: **4** (auth &
permission logic — rate limiting, lockout, user block all ride on the
store that changed), **7** (a dependency removed), **8** (config /
secrets handling).

## 4. Auth & permission logic

**Same store, fewer code paths.** Every environment already ran
`KV_BACKEND=memory` (commit `32c4490`, 2026-06-15), so rate limits,
login lockout, MFA temp tokens and user blocks were already process-local
and stay process-local. What changed is the *plumbing*: the string-command
`Batch("INCR k 1min", "GET k")` emulation of the shrmpl-kv wire protocol
is gone and callers use `Incr`/`Get` directly. Semantics preserved (proved
by the pytest suite: lockout, rate limit 429, user block 403 all green):

- Incr honours the TTL only on create — an attacker's repeated attempts
  cannot extend their own window (unchanged, now documented in `kv.go`).
- The "key not found" sentinel comparisons (`"*KEY NOT FOUND*"`) are gone;
  missing reads as `""`. A lock is `"1"`; a block is any non-empty value.

**Fail open → fail closed.** Four places treated a store error as "KV
outage, let the request through": `CheckRateLimit`,
`CheckRateLimitAndUserBlock`, `CheckUserBlocked`, and the lockout check in
`AuthHandler` (which had an explicit comment justifying it for a *remote*
KV). With an in-process store the only possible errors are programming
errors (closed store, key > 100 chars), so each now logs at ERROR and
responds 500. `MFAVerifyHandler` already did this. Net: a bug can no
longer silently disable rate limiting or lockout.

Key-length bound: `RPM*<subject>*<minute>` and `loginfail-user-<name>`
must stay under 100 chars. Usernames are bounded by the model validation
(well under that); IPv6 max 45 chars. Not a new limit.

**Process-local caveat, restated in `kv.go`:** two or more API processes
behind a load balancer make every limit per-process. Graduation to a
shared store is a future ticket, not a config flag any more — that is the
point of PS0003.

**Logging.** slog's text handler quotes values, so a hostile username can
no longer inject a fake log line by embedding `\n` (the old
`fmt.Fprintf("[%s] %s")` path could). Log level is DEBUG everywhere, as
before; DEBUG lines include rate-limit keys (username / client IP + minute)
and always did. No secrets are logged; `PwHash`/`TOTPSeed` never reach a
log call. `AddSource` is trimmed to `file.go:line` so build paths do not
leak into journald.

## 7. Dependency removed

`shrmpl-kv`, `shrmpl-log`, `shrmpl-vault` (three unversioned binaries
fetched by `bin/010` from a private URL) and their TCP clients are gone.
Removed attack surface: the API no longer opens outbound TCP to
`KV_HOST_PORT` / `LOG_RECEIVER_HOST_PORT`, no longer speaks a custom
unauthenticated line protocol in which key names were interpolated
unescaped into commands, and no longer sends every log line — including
usernames and client IPs — over plaintext TCP. No new dependency added;
`log/slog` is stdlib.

## 8. Config / secrets

**Vault removed.** `LoadConfig` no longer fetches `DATABASE_URL`,
`JWT_SECRET`, `TOTP_ENCRYPTION_KEY` over mTLS; every environment reads
them inline from its env file, which is what UAT/prd (`infra/*/etc`) and
e2e already did. The hard-coded
`/Users/brandon/.../shrmpl_vault_server_fullchain.pem` in `config.go` is
gone with it. Consequence: **the env file is the secret** on every host —
`root:proto 0640` per the `infra/` headers; `doc/320` already says the
same for `etc/api-srv.env`.

**`etc/loc-api-srv.env` is committed with inline local-only values.**
Same pattern `etc/e2e-api-srv.env` has used since June: the roles.sql
default DB password (already in git), a JWT secret and a TOTP key that
say "loc-dev-only … not-for-any-deployed-environment" in their value. Not
usable anywhere but a laptop. The TOTP key is 64 hex chars because the
handler `hex.DecodeString`s it — the e2e file's old value was not hex and
would have failed MFA setup in the e2e run; fixed alongside.

**Local MFA enrolments break once.** Brandon's `loc_starter_db` has TOTP
seeds encrypted under the old dev key (from the vault file). The new local
key cannot decrypt them: reset MFA locally (`POST /admin/resetUserMFA`)
or rebuild with `bin/510`. The old dev vault file is recoverable from
history (`git show 93ca148:etc/vault-configs/dev_…`) if needed once.

**Secrets in history** — unchanged from PS0002: the old dev values remain
in git history; rotation is on the PS0002 list.

**Deleted locally by this session** (all gitignored, all regenerable):
`bin/tools/*` (shrmpl binaries), `etc/vault-configs/`,
`etc/crt/shrmpl_vault_*.pem`. `etc/crt/api-srv-*.pem` kept.

## What did I miss?

- `TLS_CERT_PATH`/`TLS_KEY_PATH` in `loc-api-srv.env` are now relative;
  only read when `ENABLE_TLS=true`, which is false locally. If someone
  flips it on, `bin/215` runs from the repo root so the paths resolve.
- `bin/216-stop-local-stack` does `pkill -f vite` — same as the old 211,
  and it will kill *any* vite on the machine, not just this repo's. Not new.
- Nothing here touches `infra/*/shrmpl-etc/` (shrmpl-hook, the webhook
  receiver) — a different product, intentionally untouched.
