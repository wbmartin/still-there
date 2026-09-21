# 2026-09-21 — PS0008 app_lib split

Six packages, not five: the logger has to live somewhere that neither
`app` (which holds it) nor `auth` (whose RBAC manager takes it) can
import from without a cycle, so `internal/logging` exists. Everything
else followed the ticket's table. The move was a python script: git mv,
package clauses, a symbol→package map for the handler qualifiers, then
compile errors to prune imports. TOTP crypto came out of the MFA handler
into `auth/totp.go` as exported functions plus a one-line
`ValidateTOTPCode`, so `grep aes|totp internal/handler` is clean.

revive immediately flagged `app.AppServices` and
`security.SecurityHeadersMiddleware` as stutters — renamed, mechanical.
Lint total went down, not up.
