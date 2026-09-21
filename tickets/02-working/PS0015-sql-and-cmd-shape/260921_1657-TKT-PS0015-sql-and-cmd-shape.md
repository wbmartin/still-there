# 2026-09-21 — PS0015 sql/ and cmd/ shape

The commented-out "SQL" file became a doc with real fenced SQL; the app
role got one name on both deploy paths (`<slug>_app`), which deleted two
steps from the migration runbook; fixtures lost their 015- era number;
`migrate new` exists. One slip: a `&&` chain stopped at a failed `git rm`
and `doc/340` was pointed at before it existed — caught by the rebrand
dry-run grep, fixed in the next commit.
