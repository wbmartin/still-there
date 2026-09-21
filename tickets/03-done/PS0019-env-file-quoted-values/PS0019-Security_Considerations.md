# PS0019 — Security Considerations

Branch `TKT-PS0019-env-file-quoted-values`. Profile **8** (config
parsing). Nine-line change to the env-file loader that every secret in
the app passes through.

- **Secrets unaffected.** `DATABASE_URL`, `JWT_SECRET`,
  `TOTP_ENCRYPTION_KEY` are unquoted in every env file in the repo and
  in the `SERVERONLY` placeholders, so their parsed values are
  byte-identical before and after. A host file that *had* quoted a
  secret was previously feeding the quotes into the secret; it now gets
  the intended value. If a deployed host quoted `JWT_SECRET`, existing
  sessions would be invalidated on deploy — check the UAT file before
  the next deploy (`grep '^JWT_SECRET="'`).
- **No new parsing surface.** Only a leading+trailing matching quote pair
  is stripped; no escapes, no variable expansion (the reason `godotenv`
  was declined — `${VAR}` expansion in a secrets file is a foot-gun).
- **JWT issuer** now reads `Proto App` instead of `"Proto App"`; nothing
  validates `iss`, so no session impact.

## What did I miss?

- A value like `"a"b"` becomes `a"b` — matching outer pair only. Fine.
- `cmd/migrate -config` uses the same loader; its `DATABASE_URL` is
  unquoted in `infra/*/etc/*-migrate.env`.
