# 2026-09-21 — PS0012 entity pages

The one real bug was a page that looked in a cache instead of asking the
server: a cold deep link to a user group gave an empty form. There is no
`GET /appUserGroup`, so the store reads the list endpoint with one id and
merges, the way the user store already did. Wrote the Playwright case
first, watched it fail against the old page, then fixed it. `ROUTES` are
absolute now — forty `` `/${ROUTES.X}` `` wrappers gone, and the class of
"forgot the slash" bug with them. The auth guard's `isValidating` state
was a derived value in disguise; deriving it deleted the `setTimeout`
dodge and the lint error. PS0013's in-range bumps had left four eslint
errors on `main`; all gone, zero warnings. The README *Scaffolding*
section is the point of the ticket: the next agent copies four files,
adds five lines by hand, and does not build a factory.
