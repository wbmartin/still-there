# Dev environment: daily start

One-time setup (tools, the developer database) is
doc/050-dev-env-init.md. Commands below run from the repo root unless a
step says otherwise; each section is its own shell.

## Shell 1: database
```bash
docker start loc-postgres        # create it once: doc/050-dev-env-init.md
docker exec -it loc-postgres /bin/bash
#inside the container
su - postgres
# inside postgres shell
psql
# inside postgres
\c loc_proto_db
```

After a pull that brings a migration (and the first time, if doc/050 was
skipped), bring the developer database up to date:
```bash
./bin/205-migrate-db
```

## Shell 2: API server
```bash
./bin/215-run-api-srv            # builds first (bin/100), then runs in the foreground
```
Is it up? From any shell:
```bash
curl -s http://localhost:8080/health
```
(The port is `SERVER_PORT` in `etc/loc-api-srv.env`.)

## Shell 3: web app
```bash
./bin/220-run-web-app            # Vite on http://localhost:5173, /api proxied to :8080
```

## To stop
```bash
./bin/216-stop-local-stack       # this checkout's API and Vite
```
