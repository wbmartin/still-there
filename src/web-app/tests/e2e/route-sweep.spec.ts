import { expect, test, type Page } from "@playwright/test";
import { ROUTES } from "../../constants/routes";
import { session } from "./fixtures/auth";
import {
  api,
  cookieForStorageState,
  signIn,
  type ApiSession,
} from "./fixtures/api-client";

// Every route, as every kind of visitor (B066). The other specs test chosen
// flows as admin; this one opens each page as admin, as basic (no user
// admin grants) and signed out, and fails on what a crash looks like: the
// router's or the boundary's error heading, a page with almost no text, or
// any console error or uncaught exception. A route the visitor may not see
// must land where the app sends them instead (the dashboard, or the login
// form), and that landing page is held to the same checks.
//
// Adding a route to constants/routes.ts without a row here fails the first
// test below. Fixture users: sql/fixtures/test-seed.sql (tenant 2); mfa is
// left out (its sign-in stops at the second factor; mfa.spec.ts owns it).

type Visitor = "admin" | "basic" | "anonymous";

interface Landing {
  path: string;
  heading: RegExp;
}

interface SweepRow {
  // `{exampleId}` and `{appUserId}` are filled from the fixtures at run
  // time; the ids depend on the seed's sequences.
  path: string;
  landing: Record<Visitor, Landing>;
}

const DASHBOARD: Landing = { path: ROUTES.DASHBOARD, heading: /^Dashboard$/ };
const LOGIN: Landing = { path: ROUTES.LOGIN, heading: /^Login to / };

const at = (path: string, heading: RegExp): Landing => ({ path, heading });

// The same page for everyone signed in; the login form for anyone else.
const signedIn = (landing: Landing): Record<Visitor, Landing> => ({
  admin: landing,
  basic: landing,
  anonymous: LOGIN,
});

// Admin only: basic is refused (the page's grant check sends them to the
// dashboard), anyone signed out goes to the login form.
const adminOnly = (landing: Landing): Record<Visitor, Landing> => ({
  admin: landing,
  basic: DASHBOARD,
  anonymous: LOGIN,
});

// Open to all three, signed in or not.
const everyone = (landing: Landing): Record<Visitor, Landing> => ({
  admin: landing,
  basic: landing,
  anonymous: landing,
});

const INVITE_HEADING = /^Join /;
const MFA_SETUP_HEADING = /^Set up Multi-Factor Authentication$/;

const SWEEP_ROW_LIST: SweepRow[] = [
  { path: ROUTES.LOGIN, landing: everyone(LOGIN) },
  {
    path: ROUTES.INVITE,
    landing: everyone(at(ROUTES.INVITE, INVITE_HEADING)),
  },
  {
    path: ROUTES.MFA_SETUP,
    landing: everyone(at(ROUTES.MFA_SETUP, MFA_SETUP_HEADING)),
  },
  {
    path: ROUTES.MFA_VERIFY,
    landing: everyone(at(ROUTES.MFA_VERIFY, MFA_SETUP_HEADING)),
  },
  { path: ROUTES.DASHBOARD, landing: signedIn(DASHBOARD) },
  {
    path: ROUTES.EXAMPLE_LIST,
    landing: signedIn(at(ROUTES.EXAMPLE_LIST, /^Examples$/)),
  },
  {
    path: `${ROUTES.EXAMPLE_EDIT}?exampleId=new`,
    landing: signedIn(at(ROUTES.EXAMPLE_EDIT, /^Add Example$/)),
  },
  {
    path: `${ROUTES.EXAMPLE_EDIT}?exampleId={exampleId}`,
    landing: signedIn(at(ROUTES.EXAMPLE_EDIT, /^Edit Example$/)),
  },
  {
    path: ROUTES.APP_USER_LIST,
    landing: adminOnly(at(ROUTES.APP_USER_LIST, /^User Admin$/)),
  },
  {
    path: `${ROUTES.APP_USER_EDIT}?appUserId=new`,
    landing: adminOnly(at(ROUTES.APP_USER_EDIT, /^Add User/)),
  },
  {
    path: `${ROUTES.APP_USER_EDIT}?appUserId={appUserId}`,
    landing: adminOnly(at(ROUTES.APP_USER_EDIT, /^Edit User/)),
  },
  {
    path: ROUTES.APP_USER_GROUP_LIST,
    landing: signedIn(at(ROUTES.APP_USER_GROUP_LIST, /^User Group Admin$/)),
  },
  {
    path: `${ROUTES.APP_USER_GROUP_EDIT}?appUserGroupId=new`,
    landing: adminOnly(at(ROUTES.APP_USER_GROUP_EDIT, /^Edit User Group$/)),
  },
  {
    // Group 2 (Admin) is a fixed fixture id.
    path: `${ROUTES.APP_USER_GROUP_EDIT}?appUserGroupId=2`,
    landing: adminOnly(at(ROUTES.APP_USER_GROUP_EDIT, /^Edit User Group$/)),
  },
  {
    path: ROUTES.USER_PREFERENCES,
    landing: signedIn(at(ROUTES.USER_PREFERENCES, /^Profile$/)),
  },
  {
    // The issue-report module is on in the e2e API (etc/e2e-api-srv.env).
    path: ROUTES.ISSUE_REPORT_LIST,
    landing: signedIn(at(ROUTES.ISSUE_REPORT_LIST, /^Issue reports$/)),
  },
];

// What a crash looks like: the router's errorElement (app.tsx) and the
// error boundary (components/error-boundary.tsx).
const ERROR_HEADING =
  /^(Something went wrong|This page ran into a problem)$/;
// Under this much body text the page is blank in all but name (GRC's
// smoke.mjs threshold).
const MIN_BODY_TEXT_LENGTH = 40;

const BASIC_PASSWORD = "basic123!";

interface FixtureIds {
  exampleId: number;
  appUserId: number;
}

let fixtureIds: FixtureIds | null = null;

// Looked up once, through the admin API session global setup saved.
const readFixtureIds = async (): Promise<FixtureIds> => {
  if (fixtureIds) return fixtureIds;
  const admin = session();
  const exampleList = await api.get<
    { exampleId: number; exampleText: string }[]
  >(admin, "/exampleList");
  const userList = await api.get<{ appUserId: number; userName: string }[]>(
    admin,
    "/userList",
  );
  const example = exampleList.find(
    (row) => row.exampleText === "Sample example 1",
  );
  const basic = userList.find((row) => row.userName === "basic");
  if (!example || !basic) {
    throw new Error("route sweep: fixture example or basic user missing");
  }
  fixtureIds = { exampleId: example.exampleId, appUserId: basic.appUserId };
  return fixtureIds;
};

const fillPath = (path: string, ids: FixtureIds): string =>
  path
    .replace("{exampleId}", String(ids.exampleId))
    .replace("{appUserId}", String(ids.appUserId));

// The browser logs every refused request as a console error with no URL;
// refusals are judged from the responses instead (expectedRefusalList).
const RESOURCE_ERROR = /^Failed to load resource: /;

// The only refusal a page may meet: signed out, the session check on load
// (GET /refresh) answers 401, and that is how the app learns to show the
// login form.
const expectedRefusalList: Record<Visitor, RegExp[]> = {
  admin: [],
  basic: [],
  anonymous: [/^401 GET \/api\/refresh$/],
};

/**
 * Start collecting console errors, uncaught exceptions and refused
 * requests (4xx/5xx) other than the visitor's expected ones.
 */
const watchForErrors = (page: Page, visitor: Visitor): string[] => {
  const errorList: string[] = [];
  page.on("console", (message) => {
    if (message.type() === "error" && !RESOURCE_ERROR.test(message.text())) {
      errorList.push(`console: ${message.text()}`);
    }
  });
  page.on("pageerror", (error) => {
    errorList.push(`pageerror: ${error.message}`);
  });
  page.on("response", (response) => {
    if (response.status() < 400) return;
    const refusal =
      `${response.status()} ${response.request().method()} ` +
      new URL(response.url()).pathname;
    const expected = expectedRefusalList[visitor].some((pattern) =>
      pattern.test(refusal),
    );
    if (!expected) errorList.push(`response: ${refusal}`);
  });
  return errorList;
};

/** Open `path` as `visitor`; hold the page it lands on to the checks. */
const expectRouteRenders = async (
  page: Page,
  visitor: Visitor,
  path: string,
  landing: Landing,
): Promise<void> => {
  const errorList = watchForErrors(page, visitor);
  await page.goto(path);

  await expect
    .poll(() => new URL(page.url()).pathname)
    .toBe(landing.path);
  await expect(
    page.getByRole("heading", { name: landing.heading }),
  ).toBeVisible();
  // Late failures (a data call, a lazy render) land after the heading.
  await page.waitForLoadState("networkidle");

  await expect(page.getByRole("heading", { name: ERROR_HEADING }))
    .toHaveCount(0);
  const bodyText = (await page.locator("body").innerText()).trim();
  expect(bodyText.length).toBeGreaterThan(MIN_BODY_TEXT_LENGTH);
  expect(errorList).toEqual([]);
};

test("every route in ROUTES is swept", () => {
  const sweptPathList = new Set(
    SWEEP_ROW_LIST.map((row) => row.path.split("?")[0]),
  );
  const unsweptList = Object.values(ROUTES).filter(
    (path) => !sweptPathList.has(path),
  );
  expect(unsweptList).toEqual([]);
});

test.describe("route sweep: admin", () => {
  // The saved session from global setup.
  for (const row of SWEEP_ROW_LIST) {
    test(row.path, async ({ page }) => {
      const path = fillPath(row.path, await readFixtureIds());
      await expectRouteRenders(page, "admin", path, row.landing.admin);
    });
  }
});

test.describe("route sweep: basic", () => {
  test.use({ storageState: { cookies: [], origins: [] } });

  let basicSession: ApiSession | null = null;

  test.beforeAll(async () => {
    basicSession = await signIn("basic", BASIC_PASSWORD);
    expect(basicSession.securityContext).toBe(2);
  });

  test.beforeEach(async ({ context }) => {
    if (!basicSession) throw new Error("basic did not sign in");
    await context.addCookies([cookieForStorageState(basicSession)]);
  });

  for (const row of SWEEP_ROW_LIST) {
    test(row.path, async ({ page }) => {
      const path = fillPath(row.path, await readFixtureIds());
      await expectRouteRenders(page, "basic", path, row.landing.basic);
    });
  }
});

test.describe("route sweep: signed out", () => {
  test.use({ storageState: { cookies: [], origins: [] } });

  for (const row of SWEEP_ROW_LIST) {
    test(row.path, async ({ page }) => {
      const path = fillPath(row.path, await readFixtureIds());
      await expectRouteRenders(
        page,
        "anonymous",
        path,
        row.landing.anonymous,
      );
    });
  }
});

// The detector itself: a path no route matches gets the router's error
// page, and the checks above must see it.
test("an unknown path shows the error heading", async ({ page }) => {
  await page.goto("/no-such-page");
  await expect(page.getByRole("heading", { name: ERROR_HEADING }))
    .toBeVisible();
});
