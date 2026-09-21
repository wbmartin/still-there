# PS0014 — Security Considerations

Branch `TKT-PS0014-test-harness-and-bin-numbering`. Profiles: **8**
(scripts, config, CI tooling) and **7** (dependency pins). No endpoint
or application-code change; the only Go changes outside tests are in
the load-test tool.

## 7. Dependencies

- `requests 2.31.0 → 2.32.5`: closes CVE-2024-35195 (certificate
  verification could be skipped on a reused session). Test-only, but the
  suite talks to real API instances.
- `pytest 7.4.3 → 8.4.2`. `psycopg2-binary` removed — imported in
  `conftest.py`, never used; one fewer native wheel on every host that
  runs the gate.
- No new dependency anywhere. The load test module has none.

## 8. Scripts and config

**No test script writes to the source tree.** `310` (`golines -w`) and
`320` (`eslint --fix`) are gone; a green gate can no longer be one that
reformatted the code it tested. Verified: `git status` clean after
`bin/300 all`.

**Load test.** Three things the review flagged, fixed: (1) `InsecureSkipVerify`
was unconditional — now `--insecure`, default off, so pointing it at a
real host verifies the certificate; (2) `admin`/`admin123!` were compiled
in — now `--user`/`--password` flags with those fixture values as
defaults, so a host with rotated fixtures passes its own; (3) it lived
under `cmd/` in the API module, so `go build ./...` shipped it — it is
its own module in `src/load-test/`, built only by `bin/330`. It still
logs in with a real password over whatever `TEST_BASE_URL` says; `330`
refuses to run unless `/health` answers, so a typo cannot spray
credentials at the wrong host for a minute.

**Passwords in test env files.** Still present, still local-only
(`etc/loc-test.env` docker superuser; `SERVERONLY` on the infra copy),
now written once each instead of three times — one line to rotate.

**`vite.config.ts` fails without `VITE_API_PROXY_URL`** rather than
silently proxying to `localhost:8080`; a mis-set environment now shows
itself instead of hitting a developer's stack.

**`playwright.config.ts` reads the API env file** for port and bind
address. It reads only those two keys; the file also holds local-only
secrets, which never leave the process. A build host's file is
`root:shrmpl-hook 0640`, same as before.

**`400-make-local-dev-certs`** writes a self-signed key with `chmod 600`
into a gitignored directory; refuses to overwrite without `FORCE=1`.
Documented as local-only.

**`conftest.py` `/auth2`** dead branch removed; the client now fails
loudly if a test user requires MFA instead of posting a placeholder code
to a non-existent endpoint.

## What did I miss?

- `tmp/test-reports/` HTML (pytest, load test) can contain request URLs
  and usernames from the seeded fixtures; gitignored, laptop-only, same
  content the old `tests/reports/` had.
- `300 py` copies the e2e env (with its local secrets) to
  `tmp/test-logs/py-api-srv.env` for the private instance — unchanged
  from `200`, gitignored.
