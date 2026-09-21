# 2026-09-21 — PS0018 profile page

The whole ticket hangs on one sentence: the row is the one in the JWT.
`/me` never reads an id from the body, needs only LOGON, and its UPDATE
lists four columns — so the admin surface (`PUT /user`, the 9xxx
grants) is untouched and a user cannot reach their own `mfa_exempt` or
groups through the new door. The preferences column reused the PS0016
jsonb pattern verbatim, `jsonbParam` and all, which is what that ticket
was for. On the web the interesting part was not the page but
`index.css`: the Vite template's hard-coded dark palette and its
`prefers-color-scheme: light` override would have fought a saved scheme,
so Mantine now owns body colours and the saved value is pushed through
`setColorScheme` from `app.tsx` on every session. The Playwright spec
checks the scheme on `/dashboard` after navigation, not on the page
that set it — that is the claim that matters. Screenshots taken via a
throwaway spec: light profile page, forced-dark list page, both right.
