# 2026-09-21 — PS0020 the second factor, finally tested

A coverage ticket that turned into two fixes, which is what coverage
tickets are for. The first Go test I wrote for the temp-token path —
"rate limit exceeded, repository never reached" — failed: `SkipJWT()`
was there but `CheckAllChecksComplete()` was not, so a request the
limiter had already answered kept going. The second surprise came from
asserting the temp token was burnt after a good code: it was not.
`KV.Set(key, "", "0")` is an invalid TTL, the error was thrown away, and
every temp token lived its full ten minutes — replayable. `KV.Del` had
been sitting there all along. The fixture user `mfa` starts with no seed
on purpose, so enrolment is inside the test and no environment key has
to line up; pyotp does the arithmetic in pytest, thirty lines of
`node:crypto` do it in Playwright. The lockout test runs last because it
locks the IP, which is the point of the lockout.
