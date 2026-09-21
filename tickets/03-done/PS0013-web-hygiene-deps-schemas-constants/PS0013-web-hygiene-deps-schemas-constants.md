---
id: PS0013
title: Web hygiene — unused deps, dead schemas/types, console noise, generated grants/endpoints, naming, modal API
status: in-review
blocked_by:
waiting_on:
priority: medium
domain: web
raised: 2026-09-21
raised_by: template review 2026-09-21
asked:
claimed: 2026-09-21 by TKT-PS0013-web-hygiene-deps-schemas-constants
merged: 6f0aff1
deployed: none
validated: 2026-09-21
---
# PS0013 — Web leftovers from earlier iterations

**Source:** template review 2026-09-21. Independent of PS0012 for the
deletions; the constants and modal items are easier after it.

## Dependencies (`src/web-app/package.json`)

- **Unused runtime (0 imports each):** `@mantine/dates`, `@mantine/dropzone`,
  `@mantine/hooks`, `@mantine/nprogress`, `@mantine/spotlight`,
  `@mantine/tiptap`, `dayjs`, `dompurify`, `isomorphic-dompurify`,
  `@types/dompurify` (deprecated stub; dompurify ships types).
- **Unused dev:** `msw`, `@testing-library/react`, `@testing-library/jest-dom`,
  `@testing-library/user-event` — installed, no test imports them.
- **`zod` is imported** (`schemas/form-schemas.ts:1`) **but not declared**;
  resolves only transitively via `mantine-form-zod-resolver`.
- **Drift:** Mantine 8.3.10 vs 9.6.2 (whole family), eslint 9→10,
  `@vitejs/plugin-react` 5→6, vitest 4→5, jsdom 27→29. Decide a cadence
  for a template: pin exact and bump deliberately, or renovate.

## Dead code

- `schemas/form-schemas.ts` is ~90% dead: only `loginSchema` is used
  (`login.tsx:5, 30`). `userCreate/Update`, `exampleCreate/Update`,
  `userGroupCreate/Update`, `passwordChange` schemas and all eight
  `*FormData` types are unreferenced; edit pages re-declare weaker inline
  validators (`app-user-edit-page.tsx:54-59` vs `userUpdateSchema`).
  `userGroupUpdateSchema` uses `appUserGroupID` (wrong casing);
  `exampleCreateSchema` says `exampleInt: z.number()` but the form stores
  strings. `schemas/README.md:33` has a stale "TODO: Enable when version
  compatibility resolved" and a "Migration Status" section. → Either wire
  every form through `zodResolver(schema)` and delete the inline
  validators + `types/*FormData`, or delete the file. Per-feature
  `schema.ts` (PS0012) is the natural home.
- `types/`: `ApiExampleModel`, `ApiUserModel`, `ApiAppUserGroupModel`
  (byte-identical to the non-Api interfaces), `ExampleListResponse`,
  `UserListResponse`, `shellExample` — referenced only at definition.
  `ApiResponse<T>` declared twice (`example-type.ts:34`, `user-type.ts:33`).
  `AuthState` fields duplicated into `AuthStore`. → one `types/api.ts` with
  `ApiResponse<T>` + `EntityBase`.
- `grants.getAllIds`, `grants.nameToId` unused; `isSessionExpired` used
  only in tests; `API_ENDPOINTS.GRANT_LIST`, `CHANGE_OWN_PASSWORD` unused
  because `auth-store.ts:54, 104, 152, 175, 204` and
  `app-user-store.ts:124` (`"/admin/resetUserMFA"`) hardcode paths.
- `components/PasswordGenerator.tsx:23` renders the literal text
  "Password Generator Component".
- `App.css` (untouched Vite scaffold, not imported), `public/vite.svg`.

## Error handling predates `utils/api.ts`

`auth-store.ts:59` `throw new Error("Login failed")` discards the server
message; `:180, 210-211` parse error bodies by hand instead of
`throwForFailedResponse`/`readErrorMessage`; `MFAOnboardingPage.tsx:42-50,
79-86` and `UserRegistrationModal.tsx:48-60` do raw `apiRequest` +
`response.ok` + body parse. → Move `readErrorMessage` from
`create-entity-store.ts:53` into `utils/api.ts`; add `apiJson<T>(endpoint,
init)` so no caller touches `response.ok`.

26 `console.error` calls in app code (e.g. `example-edit-page.tsx:66, 124,
152`, `auth-store.ts:154`, `use-auth-guard.ts:21`), most swallowing errors
the store already recorded; a commented-out `console.log` at
`auth-store.ts:101`. → one `reportError(err, context)`; ESLint `no-console`.

## Backend coupling

`constants/grants.ts:2-22` hardcodes grant ids; `constants/api.ts:1-23`
duplicates backend paths; nothing tests that either matches
`src/api-srv`. → Generate both from the API (PS0010 exports the grant
catalog as JSON; a `go generate` or a script writes `grants.ts` and
`api.ts`), and make `apiRequest` accept only `API_ENDPOINTS` values.

## Naming and component API

- `pages/MFAOnboardingPage.tsx` vs kebab-case siblings; `components/` are
  PascalCase files while pages/stores/hooks are kebab; `App.tsx` at root.
  → kebab for files everywhere.
- Modal prop vocabularies differ: `DeleteConfirmModal` `isOpen/onConfirm/onCancel`,
  `UserRegistrationModal` `opened/onClose/onSuccess`,
  `AddRemoveUsersFromGroupModal` default export vs named elsewhere;
  `ChangePasswordModal` takes `mode: "own"|"other"` plus its own
  loading/error props, while `app-user-edit-page.tsx:325-337` uses
  `modals.openConfirmModal` from `@mantine/modals` (already a provider in
  `main.tsx:14`). → `@mantine/modals` for confirms; one prop vocabulary;
  feature modals live in their feature folder.

## Lint/config

- `eslint.config.js:27` `max-len 80` is the only custom rule and is
  effectively decorative (long JSX lines pass). → drop it or run prettier;
  add `no-console`.
- `tsconfig.app.json` `include: ["."]` type-checks `tests/` and
  `playwright.config.ts` under the browser lib (works only because
  `@types/node` arrives via `vite/client`). → `tsconfig.test.json`.
- `test:coverage` script exists with no threshold.

## What changed and why

Branch `TKT-PS0013-web-hygiene-deps-schemas-constants`, 6 commits. Per
Brandon's direction: everything except the generator.

| Area | Change |
|---|---|
| Dependencies | 10 unused runtime + 4 unused dev packages removed; `zod` declared (exact, the version already resolved); `@mantine/hooks` kept (peer of `@mantine/core`), `@vitest/coverage-v8` kept (`test:coverage`). In-range bumps applied (Mantine 8.3.10→8.3.18 family, @tabler, types, eslint plugins). **Cadence** (README): caret ranges, `npm update --legacy-peer-deps` + `npx depcheck` on any web ticket, majors as their own ticket. `npx depcheck` → "No depcheck issue"; `npm outdated` in-range drift → 0. |
| Dead code | `schemas/form-schemas.ts` → only `loginSchema` (+ `LoginFormData`); `schemas/README.md` rewritten (3 lines). `types/api.ts` holds `ApiResponse<T>` and `EntityBase`; `ApiExampleModel`, `ApiUserModel`, `ApiAppUserGroupModel`, `ExampleListResponse`, `UserListResponse`, `shellExample`, the duplicate `ApiResponse` all deleted. `grants.nameToId`/`getAllIds` deleted. `PasswordGenerator`'s placeholder text removed. `App.css`/`vite.svg` were already gone (PS0007). |
| Error handling | `readErrorMessage` moved to `utils/api.ts`; new `apiJson<T>(url, init, { anonymous })` and `jsonBody(body)`. No caller touches `response.ok` any more: login, change-context, change-own-password, MFA setup/verify, self-registration, `postOrThrow`. `anonymous: true` on the four pre-session endpoints so a wrong password/code is an ordinary error, not a session expiry. `authenticateWithAuthToken` keeps its deliberate 401/502 branching. |
| Console | `utils/report-error.ts` `reportError(err, context)`; 26 `console.error`s replaced (contexts like "deleting example"); ESLint `no-console: error` everywhere except that file and `tests/**`. |
| Endpoints | `API_ENDPOINTS` grew `AUTH`, `REFRESH`, `LOGOFF`, `CHANGE_SECURITY_CONTEXT`, `ADMIN_RESET_USER_MFA`; the stores' string literals are gone; `ApiEndpoint` type on `postOrThrow`. `constants/api.ts` and `constants/grants.ts` open with the Go file they mirror by hand (`cmd/server/main.go`, `internal/model/grants.go`). No drift test (Brandon: skip unless cheap; it is not). |
| Naming | kebab-case for every file: `mfa-onboarding-page.tsx`, `app.tsx`, all six components. |
| Modal API | `utils/confirm-delete.tsx` → `modals.openConfirmModal` for every delete (six pages; `DeleteConfirmModal` and its per-page open/id state deleted). `ChangePasswordModal` and `AddRemoveUsersFromGroupModal` speak `opened`/`onClose`/`onSubmit`; the latter is a named export. (Subagent, diff-checked.) |
| Lint/config | `max-len` dropped (decorative), `no-console` added. `tsconfig.test.json` (node + vitest types) for `tests/`, `playwright.config.ts`, `vitest.config.ts`; `tsconfig.app.json` excludes them. vitest coverage floor 40% lines/functions over `stores/utils/hooks/constants`. |

Left for PS0012 (entity pages): the three `react-hooks/exhaustive-deps`
warnings, inline validators vs zod, page state in the URL, inline styles.

## Files modified

- `src/web-app/package.json`, `package-lock.json`, `eslint.config.js`, `tsconfig.app.json`, `tsconfig.test.json` (new), `vitest.config.ts`, `README.md`
- new: `utils/report-error.ts`, `utils/confirm-delete.tsx`, `types/api.ts`
- `utils/api.ts`, `constants/api.ts`, `constants/grants.ts`, `schemas/form-schemas.ts`, `schemas/README.md`, `types/{example,user,app-user-group}-type.ts`
- `stores/{auth,app-user,examples,create-entity}-store.ts`, `hooks/use-auth-guard.ts`
- every page and component (renamed + edited); deleted `components/delete-confirm-modal.tsx`
- `doc/810` (two path lines)

## Endpoints introduced or modified

None.

## Acceptance (met 2026-09-21)

- [x] `npx depcheck` → no unused/missing deps.
- [x] `grep -rn 'console\.' src` → only `utils/report-error.ts` (and `tests/**`, allowed).
- [x] `constants/grants.ts` / `constants/api.ts` carry a header naming the Go files they mirror. (Generated header + drift test struck per Brandon.)
- [x] `tsc`, `eslint` (0 errors), vitest 28, `bin/300-run-tests all` go=163 unit=28 e2e=3 py=63.

## Worked 2026-09-21

Complete on the branch (7 commits, at the sha on origin); gate green
(go=163 unit=28 e2e=3 py=63). **Next physical action: Brandon says
"merge"** → follow `ticket-workflow` step 4 (`git merge origin/main` on
the branch, re-run `bin/300-run-tests all`, `--no-ff` into `main`, set
`merged:`). Nothing else outstanding on this ticket.

## Deploy needs

Web build only.

## Waiting for

The next UAT deploy.

## Direction (Brandon, 2026-09-21)

Fine to proceed on: dependencies, dead code, error-handling helpers,
console noise, naming, modal API, lint/config.

**Backend coupling — decided (Brandon, 2026-09-21):** "Continue
maintaining the lists separately to minimize true coupling." So **no
generator**, no build-time link between `src/web-app/constants/*.ts` and
`src/api-srv`. Strike the "Backend coupling" section's generation
proposal. What stays from it: make the stores use `API_ENDPOINTS` rather
than hardcoded paths (`auth-store.ts:54, 104, 152, 175, 204`,
`app-user-store.ts:124`) so there is one list on the web side; a header
comment in `grants.ts` and `api.ts` naming the Go files they mirror
(`internal/model/grants.go`, `cmd/server/main.go`) so a human updates
both. A drift *test* is optional and only if it reads both files at test
time without either side importing the other — Brandon has not asked for
it; skip unless cheap.
