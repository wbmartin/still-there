# PS0008 — Security Considerations

Branch `TKT-PS0008-api-split-app-lib`. Mechanical package move; no
behaviour change by construction (`git` sees 9 renames; the only new
code is `auth.ValidateTOTPCode`, a one-line wrapper, and the exported
names on six previously unexported TOTP functions).

## What moved that matters

- **All crypto is now in `internal/auth`**: JWT sign/verify, argon2id
  password hashing, TOTP secret generation, otpauth URI, AES-GCM seed
  encryption/decryption, and the code check. The handler package no
  longer imports `crypto/aes`, `crypto/cipher`, `crypto/rand`,
  `encoding/base32`/`hex`, or `pquerna/otp/totp`. Reviewing the app's
  cryptography is now one directory.
- **`AppHelper` (the explicit per-handler check chain) moved unchanged**
  to `internal/security`. Skip/Check methods, ordering, fail-closed
  behaviour from PS0002/PS0003 — all as before; the pytest suite that
  exercises 401/403/429 and lockout is green.
- **`DBPool` left `Services`.** Handlers could reach the raw pool through
  the DI struct; none did, and now none can — repositories are the only
  path to the database.

## Exported TOTP helpers

`EncryptTOTPSeed`/`DecryptTOTPSeed` take the key as hex and are now
callable from any package. Same trust as before (only handlers call
them; the key comes from `Config.TOTPEncryptionKey`). No new callers.

## What did I miss?

- `security.ClientIP` still trusts the first `X-Forwarded-For` entry —
  the PS0002 finding, unchanged, awaiting Brandon's call.
- Lint findings dropped 53 → 50; the remaining revive items
  (receiver naming, `userIds`, context-as-argument) are PS0011's and
  untouched here on purpose.
