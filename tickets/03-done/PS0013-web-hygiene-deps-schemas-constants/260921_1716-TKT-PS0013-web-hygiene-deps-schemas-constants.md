# 2026-09-21 — PS0013 web hygiene

Fourteen packages nobody imported, one (zod) everybody used but nobody
declared. The useful structural change is small: `apiJson` with an
`anonymous` flag, so the four pre-session endpoints stop hand-rolling
`response.ok` and, more to the point, a wrong password can never fire the
session-expired listener. `reportError` is one file so the console is one
decision. `npm update` crashes on this peer set (arborist `edgesOut`
null); `--legacy-peer-deps` works and is now the documented cadence. A
subagent did the modal refactor across six pages; I diffed one page and
the helper.
