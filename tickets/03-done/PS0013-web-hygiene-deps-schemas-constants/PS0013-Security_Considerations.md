# PS0013 — Security Considerations

Branch `TKT-PS0013-web-hygiene-deps-schemas-constants`. Profiles **1**
(UI) and **7** (dependencies). No API change.

## 7. Dependencies

- **Removed** ten runtime and four dev packages that nothing imported —
  `@mantine/{dates,dropzone,nprogress,spotlight,tiptap}`, `dayjs`,
  `dompurify`, `isomorphic-dompurify`, `@types/dompurify`, `msw`, three
  `@testing-library/*`. Each was attack surface in `node_modules` and
  in the supply chain (`npm audit` scope) for no benefit. `dompurify`'s
  absence is worth stating: the app never rendered HTML from the server
  (`grep dangerouslySetInnerHTML` → nothing), so there was nothing to
  sanitise; if that changes, add it back deliberately.
- **`zod` declared** at the exact version already resolved (it was reached
  only through `mantine-form-zod-resolver`'s dependency); the login form's
  validation no longer depends on a transitive pin.
- **Cadence** (the ticket asked): caret ranges, `npm update` (in-range
  bumps) as part of any ticket that touches the web app, majors as their
  own ticket. Applied here; Mantine 9 / ESLint 10 / Vitest 5 are majors
  and were not taken.

## 1. UI

- **Errors leave one door.** `utils/report-error.ts` is the only file
  allowed to call `console.*` (ESLint `no-console` is an error elsewhere).
  Twenty-six scattered `console.error`s now pass through it, so if error
  reporting ever goes to the server (`POST /error-report` exists) it is
  one change — and one place to make sure nothing sensitive is included.
  Today it prints the error object and a context noun; no request bodies,
  no tokens.
- **`apiJson` and the session listener.** Every non-2xx response now goes
  through one path. A 401/403 raises `SessionExpiredError` and tells the
  auth store — *except* on the four endpoints reached before there is a
  session (`/auth`, `/mfaSetup`, `/mfaVerify`, `/userSelfCreate`), which
  pass `anonymous: true` so a wrong password or wrong TOTP code cannot
  flip the app to "signed out" or clear caches mid-onboarding. Previously
  these sites hand-rolled `response.ok` checks with the same effect;
  the behaviour is unchanged, now stated.
- **One endpoint list.** `apiRequest` callers use `API_ENDPOINTS`
  (`ApiEndpoint` type on the store helper); `/refresh`, `/logoff`,
  `/changeSecurityContext`, `/admin/resetUserMFA` were string literals in
  the stores. Both `constants/api.ts` and `constants/grants.ts` name the
  Go file they mirror, per Brandon's "maintain separately" decision.
- **Confirmations via `@mantine/modals`**: same user-facing behaviour
  (explicit confirm before delete / MFA reset); no security change.
- **`tsconfig.test.json`**: test code and the Playwright config are no
  longer type-checked under the browser lib with `@types/node` arriving
  by accident; an app-side import of a node API would now fail the build.

## What did I miss?

- `reportError` prints to the browser console in production builds too —
  same as before. A future `POST /error-report` sink must strip anything
  the API's error body may echo back (usernames in messages).
- Dead `schemas` deleted rather than wired: the entity edit pages still
  validate inline (PS0012).
