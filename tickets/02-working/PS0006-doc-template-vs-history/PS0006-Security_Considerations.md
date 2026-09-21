# PS0006 — Security Considerations

Branch `TKT-PS0006-doc-template-vs-history`. Docs only; no code, config
or endpoints. Nothing to assess mechanically; three notes for the record:

- **Archived reviews still name real findings.** `engineering-log/251130_…-api-server-code-review.md`
  flagged TOTP-seed exposure (fixed, PS0002) and the AppHelper pattern
  (kept by design, PS0009). The header marks them point-in-time so a
  reader does not mistake a fixed finding for an open one.
- **The regenerated `src/api-srv/README.md` now lists every endpoint and
  the explicit check sequence.** That is documentation of what the code
  does, derived from `main.go`; it neither adds nor removes exposure. It
  does make the anonymous endpoints (`/auth`, `/userSelfCreate`,
  `/error-report`, `/mfaSetup`/`/mfaVerify` temp-token step, `/health`)
  easy to find — for a reviewer as much as anyone.
- **Triage found one security-adjacent bug** (PS0019): the TOTP issuer
  carries literal quotes. Cosmetic in the authenticator; not a weakness.
  And one dead code path in `tests/conftest.py` (`/auth2`) — noted in
  PS0014, no runtime effect.

No secrets touched; no `/Users/` paths remain in `doc/` or the READMEs.

## What did I miss?

- `doc/050` still mentions `sql/010-initialcreate.sql` as history
  ("an existing database built by hand from the old…") — deliberate,
  per the ticket.
