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

A template review found unused dependencies, ~90% dead schema/type files,
ad hoc error handling that predated `utils/api.ts`, 26 scattered
`console.error` calls, hardcoded backend-mirroring constants, and
inconsistent naming/modal APIs. Brandon approved everything except
generating `grants.ts`/`api.ts` from the Go source, choosing instead to
keep them maintained separately by hand.

## What changed
- `package.json`: removed 10 unused runtime deps
  (`@mantine/{dates,dropzone,nprogress,spotlight,tiptap}`, `dayjs`,
  `dompurify`, `isomorphic-dompurify`, `@types/dompurify`) and 4 unused dev
  deps (`msw`, three `@testing-library/*`); declared `zod` at its
  already-resolved version; applied in-range bumps.
- `schemas/form-schemas.ts`: cut to only `loginSchema`/`LoginFormData`;
  `schemas/README.md` rewritten to 3 lines.
- `types/api.ts` (new): `ApiResponse<T>` + `EntityBase`; deleted dead/
  duplicate types (`ApiExampleModel`, `ApiUserModel`,
  `ApiAppUserGroupModel`, `ExampleListResponse`, `UserListResponse`,
  `shellExample`, the second `ApiResponse`).
- `utils/api.ts`: `readErrorMessage` moved in; new `apiJson<T>(endpoint,
  init, { anonymous })` and `jsonBody(body)` so no caller touches
  `response.ok` directly; `anonymous: true` on the four pre-session
  endpoints (`/auth`, `/mfaSetup`, `/mfaVerify`, `/userSelfCreate`) so a
  wrong password or code can't trip the session-expired listener.
- `utils/report-error.ts` (new): `reportError(err, context)` — the only
  file allowed `console.*` (ESLint `no-console: error` everywhere else);
  replaced 26 scattered `console.error` calls.
- `constants/api.ts`, `constants/grants.ts`: stores use `API_ENDPOINTS`
  instead of hardcoded path strings; both files carry a header naming the
  Go file they mirror (`internal/model/grants.go`, `cmd/server/main.go`)
  — no generator, per Brandon's "maintain separately" decision.
- Every file under `pages/`/`components/` renamed to kebab-case.
- `utils/confirm-delete.tsx` (new): `@mantine/modals`-based confirm used
  by all six delete sites, replacing the bespoke `DeleteConfirmModal`;
  `ChangePasswordModal`/`AddRemoveUsersFromGroupModal` standardized on
  `opened`/`onClose`/`onSubmit`.
- `eslint.config.js`: decorative `max-len` dropped, `no-console` added;
  `tsconfig.test.json` (new) isolates test/Playwright type-checking from
  the app's browser lib; vitest coverage floor 40% lines/functions over
  `stores/utils/hooks/constants`.

## Rules that survived
- `apiJson`/`jsonBody` is the only path for handling a fetch response; no
  caller touches `response.ok` directly.
- Pre-session endpoints (`/auth`, `/mfaSetup`, `/mfaVerify`,
  `/userSelfCreate`) pass `anonymous: true` so a failed attempt is an
  ordinary error, never a forced sign-out.
- `reportError` is the only file allowed `console.*`; `no-console` is an
  ESLint error elsewhere.
- `constants/api.ts`/`constants/grants.ts` are hand-maintained against the
  Go source (no generator, no drift test) — each carries a header naming
  the Go file it mirrors.
- Web dependency cadence: caret ranges, `npm update --legacy-peer-deps` +
  `npx depcheck` on any web ticket, majors get their own ticket.

## Figures at close
| | date |
|---|---:|
| packages removed (runtime + dev) | 14 |
| console.error sites consolidated | 26 |
| npx depcheck issues | 0 |
| go tests | 163 |
| vitest unit tests | 28 |
| e2e tests | 3 |
| pytest | 63 |

## Endpoints
None.

## Security
Profiles 1 (UI) and 7 (dependencies). Ten runtime and four dev packages
that nothing imported were removed, reducing supply-chain/audit surface;
`dompurify`'s removal is safe since the app never rendered server HTML
(`dangerouslySetInnerHTML` grep empty). `apiJson`'s `anonymous` flag keeps
a wrong password or TOTP code from flipping the app to signed-out
mid-onboarding. `reportError` centralizes console output to one file, so a
future server-side error sink is a one-place change and a one-place
review for leaked data. `tsconfig.test.json` stops test code and the
Playwright config being type-checked under the browser lib by accident.

## Left open
- Inline validators vs. wiring the surviving `zod` schemas through forms
  — left for PS0012.
- Page state in the URL, inline styles — left for PS0012.
- A drift test between `constants/*.ts` and the Go source was considered
  and explicitly not built (Brandon: skip unless cheap).

## Sources
- `tickets/04-archive/engineering-log/260921_1716-TKT-PS0013-web-hygiene-deps-schemas-constants.md`
- git: `6f0aff1`
