import { grants } from "./grants";
import { ROUTES } from "./routes";

// The app's navigation: the navbar (app.tsx) and the help module's
// command palette (help/command-palette.tsx) both list these, so a page
// added here appears in both. An item with a grant shows only to users
// holding it; the page checks the grant again (useAuthGuard). An item
// with a module shows only while that module is on (its flag on /me).
export interface NavItem {
  route: string;
  label: string;
  grant?: number;
  module?: NavModule;
}

// The optional modules a nav item can belong to, by their /me flag.
export interface NavModuleState {
  issueReportsEnabled: boolean;
}

export type NavModule = keyof NavModuleState;

export const NAV_ITEM_LIST: NavItem[] = [
  { route: ROUTES.DASHBOARD, label: "Dashboard" },
  { route: ROUTES.EXAMPLE_LIST, label: "Examples" },
  { route: ROUTES.APP_USER_LIST, label: "User Admin", grant: grants.CREATE_USER },
  {
    route: ROUTES.APP_USER_GROUP_LIST,
    label: "User Groups",
    grant: grants.CREATE_USER_GROUP,
  },
  {
    // The issue-report module (doc/150-issue-report-module.md).
    route: ROUTES.ISSUE_REPORT_LIST,
    label: "Issue reports",
    module: "issueReportsEnabled",
  },
  { route: ROUTES.USER_PREFERENCES, label: "Profile" },
];

/** The items a user holding grantList sees with these modules on. */
export const navItemListFor = (
  grantList: number[],
  moduleState: NavModuleState,
): NavItem[] =>
  NAV_ITEM_LIST.filter(
    (item) =>
      (item.grant === undefined || grantList.includes(item.grant)) &&
      (item.module === undefined || moduleState[item.module]),
  );
