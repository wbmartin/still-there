# 2026-09-21 — PS0009 explicit checks, recorded and tested

Wrote the rule down (doc/105 + a header comment), then tried to prove it
on the copy-me handler. Putting `ExampleHandler` on an interface and
running it through httptest with a real in-process `app.Services` took
an hour; the mock panicking on any unexpected repository call is what
makes "the repo is not reached until every check passes" a real
assertion.

Two things the test taught: GET skips CSRF by design (safe methods), so
the CSRF case has to be a DELETE — now documented rather than
rediscovered; and the login handler never ran `CheckAllChecksComplete`
at all, so its exception was the implicit kind the design exists to
forbid. Written out, no behaviour change.
