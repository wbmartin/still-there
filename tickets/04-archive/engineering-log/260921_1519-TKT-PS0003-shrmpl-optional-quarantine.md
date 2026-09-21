# 2026-09-21 — PS0003 sunset shrmpl kv/log/vault

One concrete `KV`, one slog `Logger`, vault gone. The call-site surface
was worth keeping: ~300 `app.Info(code, msg)` sites untouched, code now
`code=` on the line, and `runtime.Callers(skip+1)` reproduces the old
`runtime.Caller(skip)` frame arithmetic so `AppHelper`'s wrappers still
attribute to the handler line (test proves it).

Dropping the shrmpl-kv `Batch` string protocol removed ~130 lines of
command parser and the `*KEY NOT FOUND*` sentinel. Every "KV outage →
fail open" branch became "bug → 500".

Two things the change exposed: (1) `/auth` fed the raw username into
100-char store keys — bounded at 50 now; (2) the repo/migrate integration
suites fell back to `etc/loc-api-srv.env` — i.e. the developer's live DB —
and only the vault failure had been keeping that inert. A bare
`go test ./...` ran them against `loc_starter_db` once during this ticket
(no rows left behind, checked). Fallback removed; they skip.

Also: git deleted the local `etc/vault-configs/dev_…` on branch switch
after PS0002 untracked it. Restored from `93ca148` long enough to confirm
the local DB password is the roles.sql default, then removed with the rest.
