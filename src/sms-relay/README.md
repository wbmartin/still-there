# sms-relay

One inbound-text webhook for several apps. Telnyx posts each text to
`https://webhook.teamtaggle.com/telnyx-inbound/<queue>`; the relay checks
the Ed25519 signature, stores the text in its own Postgres table
(`inbox_message`) under that queue and answers 200. Each app reads its
queue over HTTP: peek, receive (a lease), ack. Delivery is at least once,
so an app de-duplicates by message id. Outbound texts stay in each app.

Its own Go module (stdlib `net/http` + pgx), importing nothing from
`src/api-srv`: it moves to its own repository later.

## Run

```
sms-relay migrate <env file>   apply the embedded migrations (DDL role)
sms-relay serve <env file>     the webhook and the consumer API
```

`serve` refuses to start while a migration is pending or a migration
file changed after it was applied (checksums in `relay_schema_migration`).
Build for prd-wbm: `bin/140-build-sms-relay` (`dist/sms-relay`).

## Config

One flat `KEY=VALUE` file, every key required, no defaults:

- `DATABASE_URL` — the relay's DML role (`migrate` reads only this key,
  from a file with the DDL role's URL).
- `SERVER_BIND_ADDRESS`, `SERVER_PORT` — `127.0.0.1` and `8095` behind
  nginx on prd-wbm.
- `TELNYX_PUBLIC_KEY` — the account's webhook public key (base64).
- `CONSUMER_TOKEN_LIST` — `<queue>:<sha256 hex of token>,...`. A token
  reads only its queue; a queue may list two tokens while one is rotated.
  Make one: `openssl rand -hex 32` is the token (the app's secret), and
  `printf %s <token> | sha256sum` the hash.
- `RETAIN_DAYS` — acked messages older than this are deleted (daily).
- `LOG_LEVEL` — `debug`, `info`, `warn` or `error`.

## API

A queue name is `^[a-z][a-z0-9-]{0,31}$`. Bodies are capped at 64 KiB.

- `POST /telnyx-inbound/{queue}` — the webhook. Bad or stale (over 5
  minutes) signature: 401. A `message.received` event is stored; any
  other event is answered 200 and dropped. Answers `{}`.
  `POST /telnyx-inbound` (no queue) is queue `stillthere`.
- `GET /health` — 200 `{"status":"ok"}` while the database answers.

The consumer routes need `Authorization: Bearer <token>` for that queue
(401 otherwise). A message is `{id, receivedTs, from, to, text}`, oldest
first.

- `GET /queue/{queue}/peek?limit=N` — up to N (default 10, max 100)
  messages not done and not under a live lease. Changes nothing.
- `POST /queue/{queue}/receive {limit, leaseSec}` — leases up to `limit`
  (default 10, max 100) of those for `leaseSec` (5 to 600) seconds under
  one new lease id: `{leaseId, messageList}`. Rows another receive holds
  are skipped (`FOR UPDATE SKIP LOCKED`).
- `POST /queue/{queue}/ack {leaseId, idList}` — marks done the ids that
  lease still holds (not expired): `{ackCount}`. An id not acked before
  its lease ends is received again.

## Tests

`bin/300-run-tests go` covers the module. The database tests build
their own scratch database, `loc_smsrelay_test_db`, on the gate's
Postgres (`TEST_PG_URL_BASE`, `etc/loc-test.env`) and skip without it.

## Deploy

`doc/320-prd-wbm-deploy.md`, SMS relay: user `svc-smsrelay`, root
`/srv/prd-smsrelay`, unit `prd-smsrelay.service`, database
`prd_smsrelay_db` (`infra/sql/smsrelay-host-db.sql`).
