# PS0012 — Security Considerations

Branch `TKT-PS0012-web-entity-pages-and-routing`. Web app only; no Go, no
SQL, no config. Profiles: **1. UI feature**, **4. Auth & permission logic**
(the auth guard hook and a render-time grant read changed).

## 1. UI feature

- **New data on screen:** none. The group edit page now fetches the group
  it already rendered when reached from the list; the fetch goes through
  the existing `GET /appUserGroupList?appUserGroupIdList=N`, which runs
  the full handler chain (`CheckJWT` → rate limit → `CheckCSRF` →
  `CheckAuthorization(GrantUserGroupRead)` → `CheckAllChecksComplete`)
  and is scoped by the caller's security context. A deep link to another
  tenant's id returns nothing and the form stays empty.
- **Input from the URL:** `appUserGroupId`, `appUserId`, `exampleId`,
  `page`, `sort`, `dir`. The ids are `Number()`-parsed (NaN → no fetch);
  `page` is `Math.max(1, Number(…) || 1)`; `sort`/`dir` only select a
  comparator and were already URL-borne. Nothing from the URL is
  rendered as HTML.
- **XSS:** no `dangerouslySetInnerHTML`, no new string interpolation
  into markup. The QR `<img>` became Mantine `<Image>` with the same
  `data:` src from the API.
- **Client-side secrets/config:** none added. `ROUTES` are paths.

## 4. Auth & permission logic

- **`hooks/use-auth-guard.ts`:** the redirect and grant decisions are
  unchanged; only the loading flag moved from state set inside the
  effect to a value derived from `authState` (`TBD` off the login page →
  `CHECKING`). The `NO` branch still saves the destination and sends to
  `/login`; the `YES` branch still requires **all** `requiredGrants` and
  bounces to the dashboard otherwise. Fails closed: any state other than
  `YES` renders "Authentication required" in every page.
- **Routes absolute:** the guard's `pathname !== ROUTES.LOGIN` comparison
  is now a like-for-like string compare; before PS0002 it never matched.
  No route became reachable or unreachable — `app.tsx` registers the
  same eleven paths.
- **Load effects gated on `authState === "YES"`:** group and example
  edit pages no longer fire their fetch before the session is known.
  The server would have rejected it anyway; this removes a spurious 401.
- **`grantList` via selector** (`app-user-edit-page`): the Reset-MFA
  button now re-renders when grants change (e.g. security-context
  switch) instead of showing a stale decision. The server still enforces
  `RESET_MFA_TOKEN` independently; the button is a convenience, not the
  check.
- **Persona coverage:** e2e runs as the seeded admin. Unauthorised
  personas are covered by the pytest suite against the API (`403` on
  missing grants), which is where the enforcement lives.

## What did I miss?

- `?page=` is user-controlled but only slices a client-side array.
- The `.sortable-column` CSS class and Mantine props replace inline
  styles; no style injection path (values are static).
- `doc/810` and the README now tell a derived app to gate load effects
  on `"YES"` and fetch by id — the secure default is the documented one.
- Nothing to rotate, no new dependency, no endpoint change.
