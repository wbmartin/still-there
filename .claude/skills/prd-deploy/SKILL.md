---
name: prd-deploy
description: Ship this app to its one server, prd-wbm (build the Go API, migrator and web app on the laptop, upload the bundle, hand the staff member the one sudo command that migrates, swaps with .prev kept and health-checks, then verify what is live). Use when asked to "deploy", "ship to prd", "release", "what is live", "health check the server", or to roll a release back. The full write-up is doc/320-prd-wbm-deploy.md.
---

# prd deploy

One laptop script and two host scripts; this skill is the procedure
around them. Settings: `etc/prd-deploy.env` (`PRD_SSH_HOST`, `APP_ROOT`,
`SITE_URL`) — read it first; the commands below use its values.

| | |
|---|---|
| `bin/600-deploy-prd` | laptop: refuses a dirty tree → builds `api-srv` + `stillthere-migrate` (linux/amd64, CGO off) + `web-pub.tar.gz` + `stillthere-api.revision` + `SHA256SUMS` → scp to `$APP_ROOT/incoming/<bundle>/` → prints the sudo command. `--check` is read-only: live revision, unit state, public `/api/health` |
| `infra/prd-wbm/bin/prd-stillthere-install` | host, as root: checksums → migrate verify / dump / up → swap with `.prev` → restart → health; unhealthy rolls itself back |
| `infra/prd-wbm/bin/prd-stillthere-rollback` | host, as root: swap live and `.prev` (code and web, never the schema) → restart → health |

**The staff member runs every host step.** sudo on the host needs their
password, so Claude never runs the install or the rollback; it hands over
the exact command and verifies afterwards.

## 1. Build and upload (Claude)

```bash
cd <repo>
git status --short        # must be empty; the script refuses otherwise
git log -1 --oneline      # the commit that will ship; confirm the branch
bin/600-deploy-prd 2>&1 | tail -12
```

Ship what the user named (normally `main` after a merge). Run
`bin/300-run-tests all` first unless it just passed on this commit.

## 2. Hand-off (one line)

The script's last lines are the command, for example:

```
ssh -t prd-wbm sudo /srv/prd-stillthere/bin/prd-stillthere-install 20261016T201500Z-e1114b8c0d2a
```

Say which commit it ships and whether migrations are pending
(`git diff --stat <live commit>..HEAD -- src/api-srv/sql/migrations`, the
live commit from `--check`). A release with a migration is forward-only:
say so.

## 3. Verify after the staff member says "done"

```bash
bin/600-deploy-prd --check
```

Pass = the revision's `commit=` is the shipped commit, the unit `active`,
`/api/health` answers `"status":"ok"`, and the script exits 0. Report it
as a short table. Claude cannot sign in to the UI; the browser check is
the staff member's.

## Rollback

Hand over `ssh -t prd-wbm sudo /srv/prd-stillthere/bin/prd-stillthere-rollback`, then
`--check`. It swaps, so running it twice puts the newer release back. It
never undoes a migration: if the older code cannot run on the new schema,
the way back is the pre-migrate dump (doc/320, Rollback) — the staff
member's call.

## Gotchas

- Working directory resets between turns: always `cd` into the repo.
- The upload fails with "not writable by this login" until the staff
  login is in the deploy group (doc/320, first install).
- Old bundles stay in `$APP_ROOT/incoming/`; the staff member prunes them.
- prd-wbm also runs gsd-notes (its own site, API on 8088, database and
  fail2ban jail). Nothing in this procedure touches them; this app is
  `/srv/prd-stillthere`, port 8090.
