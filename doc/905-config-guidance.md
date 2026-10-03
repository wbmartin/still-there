# Guidance for config files

- use the simple env file format
- don't worry about environment variables. The config file has the value you
  need
- one flat `KEY=VALUE` file per host, passed to the binary as its first
  argument. The API (`cmd/server`) loads it with `config.LoadConfig`;
  `etc/loc-api-srv.env` is the dev one. `cmd/migrate` reads only
  `DATABASE_URL` from a file of the same shape (`-config`, e.g.
  `infra/<host>/etc/*-migrate.env`)
- a key the API needs is **present in the file**. `config.LoadConfig`
  refuses to start, naming the key, when one is missing rather than guessing
  a value: `DATABASE_URL`, `JWT_SECRET`, `TOTP_ENCRYPTION_KEY`,
  `SERVER_NAME`, `APP_NAME`, `SERVER_PORT`, `SERVER_BIND_ADDRESS`,
  `ENABLE_TLS`, `JWT_SESSION_EXPIRE_HOURS`, the two `MAX_*_REQ_PER_MIN`
  keys, the three
  `FAILED_LOGIN_*` keys, `LOG_LEVEL`, and the TLS paths when
  `ENABLE_TLS=true`. A silent default is how a server ends up with every JWT
  born expired (0 hours) or every request refused 429 (a limit of 0), or an
  API meant to sit behind nginx answering on every interface. It
  also checks shape at load, not at first use: the port, hours, limits and
  threshold are whole numbers above 0, the two `FAILED_LOGIN_*_DURATION`
  keys are KV durations above 0 (`30s`, `5min`, `1h`), and `LOG_LEVEL` is
  `debug`, `info`, `warn` or `error`. The optional keys
  (`TRUSTED_PROXY_COUNT`, `CORS_ALLOWED_ORIGINS`) and the whole list are in `src/api-srv/README.md`
- a deploy never rewrites a host's env file, so a key that became required
  after the file was written must be added by hand before the new binary
  starts (`doc/320-presales-uat-deploy.md`)
- **quoting**: the parser (`config.ReadEnvFile`) splits each line on the
  **first** `=`, trims whitespace from the key and the value, and then strips
  **exactly one** matching pair of surrounding quotes, `"…"` or `'…'`. That
  is the whole grammar, and it is how a value with spaces is written
  (`APP_NAME="Proto App"`). Everything else in the value is literal:
  - no `export ` prefix (it becomes part of the key, so the key is not found)
  - no `${VAR}` or `$VAR` expansion, and no `\n` / `\"` escapes: this file
    holds secrets, and a password containing `$`, `\` or `#` has to reach the
    binary exactly as written
  - no inline comments: a `#` after the value is part of the value. Only a
    line whose first non-space character is `#` is a comment
  - a line with no `=` is skipped silently
  - a value quoted at one end only, or opening and closing with different
    quote characters, keeps those characters (`APP_NAME="Proto` gives the
    value `"Proto`, quote included; it is not an error); `""` is the empty
    string, and `""x""` strips one pair to `"x"`
  - a repeated key takes the last value in the file
- **before starting a binary built after PS0019 (`2b4f0e4`) against an env
  file written before it**, grep the host file for a quoted value: it used to
  be read *with* its quotes, so it changes value under the current parser.

  ```bash
  grep -n '="' /path/to/host.env     # double-quoted values
  grep -n "='" /path/to/host.env     # single-quoted values
  ```

  `JWT_SECRET` and `TOTP_ENCRYPTION_KEY` are the two that hurt: a quoted
  `JWT_SECRET` invalidates every live session, and a quoted
  `TOTP_ENCRYPTION_KEY` stops existing TOTP seeds from decrypting. Either
  unquote the value **and** keep the bytes the old binary saw (if the quotes
  really were part of the secret, double them: `JWT_SECRET=""abc""` parses
  as `"abc"`) or accept the re-key. `APP_NAME` is the one quoted value in
  `etc/loc-api-srv.env`, and losing its quotes is the point of the change
- the only environment variable the Go code reads is `TEST_DATABASE_URL`, and
  only the database-backed Go tests read it: they skip without it so that
  running `go test` never writes to a database the config file happened to
  name (`bin/300-run-tests go` sets it)
