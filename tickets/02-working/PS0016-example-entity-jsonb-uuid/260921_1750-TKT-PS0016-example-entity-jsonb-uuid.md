# 2026-09-21 — PS0016 jsonb and uuid on example

Two columns, six query sites, and the interesting decisions were all
about what *not* to let happen. The uuid is the server's: set once in
the create handler, absent from every SET list, and `RETURNING`ed from
the update so the response says what the row says rather than what the
client sent. No uuid library — twenty lines of `crypto/rand` in `model`
(it had to be there: repository tests import it, and handler → repository
→ handler would cycle). The jsonb trap is `json.RawMessage(nil)`: pgx
marshals it to the literal `null`, so a tiny `jsonbParam` turns absent
into SQL NULL, and the repo test checks nil → NULL → nil explicitly.
The web side is a validated textarea and a read-only field. The e2e
spec creates, edits and deletes its own row because the smoke test
counts the fixtures. One thing found on the way: the local dev database
was never stamped by the migrator, so applying 0002 there is a hand step
with the DDL role before the next presales bundle.
