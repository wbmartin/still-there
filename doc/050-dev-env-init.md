# Create the database

Requires a Postgres listening on 127.0.0.1:5432 whose `postgres` role can
create databases (the `loc-postgres` container from doc/060-dev-env-start.md,
or a native server). The schema is applied by the migrator, never by hand.

Fastest path -- a scratch database, rebuilt from nothing every time:

```bash
bin/510-reset-test-db                    # -> loc_starter_test_db
bin/510-reset-test-db loc_starter_e2e_db # any *_test_db / *_e2e_db name
```

That runs, in order: `CREATE DATABASE`, `src/api-srv/sql/roles.sql` (cluster
roles, idempotent), `go run ./cmd/migrate ... up` (the embedded
`sql/migrations/`), then `sql/fixtures/015-initialdata.sql` (seed groups,
`admin`/`admin123!`, `basic`/`basic123!`, two example rows). See the header of
that script for `PG_CONTAINER` / `PSQL` / `PGPASSWORD` / `PROTO_PG_URL_BASE`.

Manual path, for a database with a name of your choosing (e.g. the
`loc_starter_db` the local vault's `DATABASE_URL` points at):

```bash
psql -h 127.0.0.1 -U postgres -d postgres -c "CREATE DATABASE loc_starter_db;"
psql -h 127.0.0.1 -U postgres -d postgres -v ON_ERROR_STOP=1 -f src/api-srv/sql/roles.sql
cd src/api-srv
go run ./cmd/migrate -url "postgres://postgres:<pw>@127.0.0.1:5432/loc_starter_db" up
#   or, when the env file carries DATABASE_URL for a role that owns the schema:
#   go run ./cmd/migrate -config ../../etc/loc-api-srv.env up
psql -h 127.0.0.1 -U postgres -d loc_starter_db -v ON_ERROR_STOP=1 -f sql/fixtures/015-initialdata.sql
go run ./cmd/migrate -url "postgres://postgres:<pw>@127.0.0.1:5432/loc_starter_db" status
#   expect: 0001  initial_schema   applied <now> by postgres
```

An existing database built by hand from the old `sql/010-initialcreate.sql`
already has the 0001 schema: bring it under the migrator with
`go run ./cmd/migrate -url ... stamp 1` instead of `up`.

For a deployed environment, the DDL owner role and default privileges are a
one-time host step: `src/api-srv/sql/ddl-role-host-setup.sql`.


# Nginx setup
wil mayneed to have a container running nginx.  This will depend on how we solve the file upload/download problem


# Setting up certificates for local shrmpl vault (non MTLS).

1. register a domain with your name registrar (godaddy, namecheap, etc)
2. create a subdomain for your config server, e.g. config.yourdomain.com and A record it to your public ip address
  ```
   e.g. loc-proto // 155.138.163.197 
  ```
3. shell to a cloud server, sudo up, request a certificate with certbot.
  ```
  certbot -d loc-proto.cnfg.li
  ```
4. cat the cert and key files and copy them to your local machine
  ```
  cat /etc/letsencrypt/live/loc-proto.cnfg.li/fullchain.pem
  cat /etc/letsencrypt/live/loc-proto.cnfg.li/privkey.pem
  ```
and paste them in
  ```
  ~/Code/ProtoStack-ReactGo/etc/crt/shrmpl_vault_server_fullchain.pem
  ~/Code/ProtoStack-ReactGo/etc/crt/shrmpl_vault_server_privkey.pem
  ```

5. Update the ~/Code/ProtoStack-ReactGo/etc/loc-api-srv.env
  ```
VAULT_SERVER=https://loc-proto.cnfg.li:7474
  ```

6. Update your local /etc/hosts files
  ```
127.0.0.1       loc-proto.cnfg.li
  ```
7. If using the shrmpl-vault, you need to run bin/410-regenerate-vault-mtls-cters
8. Restart the shrmpl-vault server
9. restart the api-srv
