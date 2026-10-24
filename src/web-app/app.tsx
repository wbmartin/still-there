import {
  createBrowserRouter,
  RouterProvider,
  Outlet,
  Link,
  useNavigate,
} from 'react-router-dom'
import {
  AppShell,
  Burger,
  Group,
  Loader,
  Menu,
  NavLink,
  Button,
  ScrollArea,
  Stack,
  Alert,
  Text,
  Title,
  UnstyledButton,
  useMantineColorScheme,
} from '@mantine/core'
import { useDisclosure } from '@mantine/hooks'
import {
  IconChevronDown,
  IconLogout,
  IconMessageReport,
  IconSettings,
  IconUserCircle,
} from '@tabler/icons-react'
import { lazy, Suspense, useEffect, useRef } from 'react'
import { useProfileStore } from './stores/profile-store'
import { useApiStatusStore } from './stores/api-status-store'
import { usePreference } from './hooks/use-preference'
import { usePinnedPageHeader } from './hooks/use-pinned-page-header'
import { reportError } from './utils/report-error'
import Login from './pages/login'
import Dashboard from './pages/dashboard'
import ExampleListPage from './pages/example-list-page'
import ExampleEditPage from './pages/example-edit-page'
import AppUserListPage from './pages/app-user-list-page'
import AppUserEditPage from './pages/app-user-edit-page'
import AppUserGroupListPage from './pages/app-user-group-list-page'
import AppUserGroupEditPage from './pages/app-user-group-edit-page'
import UserPreferencesPage from './pages/user-preferences-page'
import MFAOnboardingPage from './pages/mfa-onboarding-page'
import InvitePage from './pages/invite-page'
import DeviceListPage from './pages/device-list-page'
import DeviceEditPage from './pages/device-edit-page'
import BeaconEditPage from './pages/beacon-edit-page'
import ContactEditPage from './pages/contact-edit-page'
import AlertListPage from './pages/alert-list-page'
import TenantSettingPage from './pages/tenant-setting-page'
import { useIssueReportStore } from './stores/issue-report-store'
import HelpButton from './help/help-button'
import { useHelpEnabled } from './help/help-preference'

import { useAuthStore, setRouteNavigator } from './stores/auth-store'
import { ROUTES } from './constants/routes'
import { navItemListFor, type NavItem } from './constants/nav'
import { APP_NAME } from './constants/app'

// The issue-report module (doc/150-issue-report-module.md) loads on first
// use: its markdown renderer and screen capture stay out of the main
// bundle, and an app with the module off never fetches them.
const IssueReportListPage = lazy(
  () => import('./pages/issue-report-list-page'),
)
const IssueReportDrawer = lazy(
  () => import('./components/issue-report-drawer'),
)

// The help module (doc/151-help-module.md) loads on first use too: the
// panel, tours and command palette stay out of the main bundle, and an
// app with help off never fetches them.
const HelpShell = lazy(() => import('./help/help-shell'))
const HelpInsightsPage = lazy(() => import('./pages/help-insights-page'))

// Simple error page for route errors
function ErrorPage() {
  return (
    <div style={{ padding: '2rem', textAlign: 'center' }}>
      <h1>Something went wrong</h1>
      <p>
        Please try refreshing the page or contact support if the problem
        persists.
      </p>
    </div>
  );
}

// Shown in place of the routed page when the session check could not reach
// the API. Distinct from the login redirect on purpose: an outage is not an
// expired session, and typing credentials at a server that cannot answer
// would not help.
function ApiUnreachableNotice() {
  const { error, loading, authenticateWithAuthToken } = useAuthStore();
  return (
    <Alert color="orange" title="Cannot reach the server" role="alert">
      <Stack gap="sm" align="flex-start">
        <Text size="sm">
          This is a connection problem, not a sign-in problem. Nothing you
          have saved is affected. Try again in a moment.
        </Text>
        {error && (
          <Text size="xs" c="dimmed">
            {error}
          </Text>
        )}
        <Button
          size="xs"
          loading={loading}
          onClick={() => void authenticateWithAuthToken()}
        >
          Try again
        </Button>
      </Stack>
    </Alert>
  );
}

// Shown above the routed page, while signed in, from the first call the API
// did not answer (a failed fetch, 5xx or 429) until the next one it does.
// One banner however many calls failed; the session is not touched.
function ApiUnreachableBanner() {
  const reason = useApiStatusStore((state) => state.reason)
  return (
    <Alert
      color="orange"
      title="The server is not answering"
      role="status"
      mb="md"
    >
      <Text size="sm">
        You are still signed in. Parts of this page may be empty or out of
        date; this notice clears itself as soon as the server answers again.
      </Text>
      {reason && (
        <Text size="xs" c="dimmed">
          {reason}
        </Text>
      )}
    </Alert>
  )
}

const router = createBrowserRouter([
  {
    path: '/',
    element: <AppLayout />,
    errorElement: <ErrorPage />,
    children: [
      {
        index: true,
        element: <Login />,
        errorElement: <ErrorPage />,
      },
        {
          path: ROUTES.LOGIN,
          element: <Login />,
          errorElement: <ErrorPage />,
        },
        {
          path: ROUTES.MFA_SETUP,
          element: <MFAOnboardingPage />,
          errorElement: <ErrorPage />,
        },
        {
          path: ROUTES.MFA_VERIFY,
          element: <MFAOnboardingPage />,
          errorElement: <ErrorPage />,
        },
        {
          path: ROUTES.INVITE,
          element: <InvitePage />,
          errorElement: <ErrorPage />,
        },
        {
          path: ROUTES.DASHBOARD,
          element: <Dashboard />,
          errorElement: <ErrorPage />,
        },
        {
          path: ROUTES.EXAMPLE_LIST,
          element: <ExampleListPage />,
          errorElement: <ErrorPage />,
        },
           {
             path: ROUTES.EXAMPLE_EDIT,
             element: <ExampleEditPage />,
             errorElement: <ErrorPage />,
           },
          {
            path: ROUTES.APP_USER_LIST,
            element: <AppUserListPage />,
            errorElement: <ErrorPage />,
          },
          {
            path: ROUTES.APP_USER_EDIT,
            element: <AppUserEditPage />,
            errorElement: <ErrorPage />,
          },
          {
            path: ROUTES.APP_USER_GROUP_LIST,
            element: <AppUserGroupListPage />,
            errorElement: <ErrorPage />,
          },
           {
             path: ROUTES.APP_USER_GROUP_EDIT,
             element: <AppUserGroupEditPage />,
             errorElement: <ErrorPage />,
           },
           {
             path: ROUTES.DEVICE_LIST,
             element: <DeviceListPage />,
             errorElement: <ErrorPage />,
           },
           {
             path: ROUTES.DEVICE_EDIT,
             element: <DeviceEditPage />,
             errorElement: <ErrorPage />,
           },
           {
             path: ROUTES.BEACON_EDIT,
             element: <BeaconEditPage />,
             errorElement: <ErrorPage />,
           },
           {
             path: ROUTES.CONTACT_EDIT,
             element: <ContactEditPage />,
             errorElement: <ErrorPage />,
           },
           {
             path: ROUTES.ALERT_LIST,
             element: <AlertListPage />,
             errorElement: <ErrorPage />,
           },
           {
             path: ROUTES.TENANT_SETTING,
             element: <TenantSettingPage />,
             errorElement: <ErrorPage />,
           },
           {
             path: ROUTES.USER_PREFERENCES,
             element: <UserPreferencesPage />,
             errorElement: <ErrorPage />,
           },
           {
             path: ROUTES.ISSUE_REPORT_LIST,
             element: (
               <Suspense fallback={<Loader />}>
                 <IssueReportListPage />
               </Suspense>
             ),
             errorElement: <ErrorPage />,
           },
           {
             path: ROUTES.HELP_INSIGHTS,
             element: (
               <Suspense fallback={<Loader />}>
                 <HelpInsightsPage />
               </Suspense>
             ),
             errorElement: <ErrorPage />,
           },
    ],
  },
])

setRouteNavigator((path) => void router.navigate(path))

const HEADER_HEIGHT = 56;

// One navbar row: a page, or a group heading over its pages.
type NavEntry =
  | { group?: undefined; item: NavItem }
  | { group: string; itemList: NavItem[] }

// Fixed header, a navbar whose link list scrolls on its own (a long list or
// a short window never pushes it off screen), and the routed page in main,
// whose title block stays pinned under the header (.app-page in index.css).
// Below `sm` the navbar is a drawer opened by the burger. Signed out (login,
// MFA) the shell is disabled: no header, no navbar.
function AppLayout() {
  const {
    logout,
    authState,
    grantList,
    securityContext,
    userName,
  } = useAuthStore();
  const navigate = useNavigate();
  const readProfile = useProfileStore((state) => state.readProfile);
  const apiUnreachable = useApiStatusStore((state) => state.unreachable);
  const profileLoaded = useProfileStore((state) => state.profile !== null);
  // The server's ISSUE_REPORTS_ENABLED, from /me: the Report button, the
  // drawer and the nav link exist only when it is on.
  const issueReportsOn = useProfileStore(
    (state) => state.profile?.issueReportsEnabled === true,
  );
  const openIssueReportDrawer = useIssueReportStore(
    (state) => state.openDrawer,
  );
  // The server's HELP_ENABLED, from /me: the Help button, the panel, the ?
  // key and the command palette exist only when it is on.
  const helpOn = useHelpEnabled();
  const { colorScheme } = usePreference();
  const { setColorScheme, clearColorScheme } = useMantineColorScheme();
  const wasSignedIn = useRef(false);
  const [navOpened, { toggle: toggleNav, close: closeNav }] =
    useDisclosure();
  // Always mounted, whatever the route or session state, so the hook's one
  // effect run sees every page that renders into it.
  const pageRef = useRef<HTMLDivElement>(null);
  usePinnedPageHeader(pageRef);

  // The profile (and with it the preferences) follows the session: loaded
  // when it is known, again after a security-context switch. The caches
  // are cleared on sign-out, so a new session starts from defaults.
  useEffect(() => {
    if (authState === 'YES') {
      void readProfile().catch((err) => reportError(err, 'profile'));
    }
  }, [authState, securityContext, readProfile]);

  // Mantine keeps the scheme in localStorage, so a reload starts from the
  // one last applied. Until /me answers, usePreference holds only the
  // default; writing that would flash it over the stored scheme. So the
  // preference applies once the profile is loaded, and the stored scheme
  // is cleared only when a session ends, not on a cold load signed out.
  useEffect(() => {
    if (profileLoaded) setColorScheme(colorScheme);
  }, [profileLoaded, colorScheme, setColorScheme]);

  useEffect(() => {
    if (authState === 'YES') {
      wasSignedIn.current = true;
    } else if (authState === 'NO' && wasSignedIn.current) {
      wasSignedIn.current = false;
      clearColorScheme();
    }
  }, [authState, clearColorScheme]);

  const handleLogout = async () => {
    closeNav();
    await logout();
    void navigate(ROUTES.LOGIN);
  };

  // Closing on tap (not on route change) also closes the phone drawer when
  // the tapped link is the page already shown. The list is
  // constants/nav.ts, filtered by the session's grants and the modules on.
  const navLink = (to: string, label: string) => (
    <NavLink
      key={to}
      component={Link}
      to={to}
      label={label}
      onClick={closeNav}
    />
  );

  // A grouped item is drawn under its group's heading, which takes the
  // place of the group's first item.
  const navEntryList: NavEntry[] = []
  for (const item of navItemListFor(grantList, {
    issueReportsEnabled: issueReportsOn,
  })) {
    const groupEntry = navEntryList.find(
      (entry) => entry.group !== undefined && entry.group === item.group,
    )
    if (groupEntry?.group !== undefined) groupEntry.itemList.push(item)
    else if (item.group !== undefined)
      navEntryList.push({ group: item.group, itemList: [item] })
    else navEntryList.push({ item })
  }

  return (
    <AppShell
      header={{ height: HEADER_HEIGHT }}
      navbar={{
        width: 240,
        breakpoint: 'sm',
        collapsed: { mobile: !navOpened },
      }}
      padding="md"
      disabled={authState !== 'YES'}
    >
      <AppShell.Header>
        <Group h="100%" px="md" justify="space-between" wrap="nowrap">
          <Group wrap="nowrap">
            <Burger
              opened={navOpened}
              onClick={toggleNav}
              hiddenFrom="sm"
              size="sm"
              aria-label="Toggle navigation"
            />
            <Title order={4}>{APP_NAME}</Title>
          </Group>
          <Group gap="sm" wrap="nowrap">
            {issueReportsOn && (
              <Button
                variant="light"
                size="xs"
                leftSection={<IconMessageReport size={16} />}
                onClick={openIssueReportDrawer}
                title="Report a bug, polish or feature request"
                data-capture-ignore
              >
                Report
              </Button>
            )}
            {helpOn && <HelpButton />}
            <Menu shadow="md" width={200} position="bottom-end">
              <Menu.Target>
                <UnstyledButton aria-label="Account menu">
                  <Group gap={4} wrap="nowrap">
                    <IconUserCircle size={18} />
                    <Text size="sm">{userName}</Text>
                    <IconChevronDown size={14} />
                  </Group>
                </UnstyledButton>
              </Menu.Target>
              <Menu.Dropdown>
                <Menu.Item
                  component={Link}
                  to={ROUTES.USER_PREFERENCES}
                  leftSection={<IconSettings size={14} />}
                  onClick={closeNav}
                >
                  Profile
                </Menu.Item>
                <Menu.Divider />
                <Menu.Item
                  leftSection={<IconLogout size={14} />}
                  onClick={() => void handleLogout()}
                >
                  Logout
                </Menu.Item>
              </Menu.Dropdown>
            </Menu>
          </Group>
        </Group>
      </AppShell.Header>
      {authState === 'YES' && issueReportsOn && (
        <Suspense fallback={null}>
          <IssueReportDrawer />
        </Suspense>
      )}
      {authState === 'YES' && helpOn && (
        <Suspense fallback={null}>
          <HelpShell />
        </Suspense>
      )}
      <AppShell.Navbar p="xs">
        <AppShell.Section grow component={ScrollArea} type="auto">
          <Stack gap={2}>
            {navEntryList.map((entry) =>
              entry.group === undefined ? (
                navLink(entry.item.route, entry.item.label)
              ) : (
                <NavLink
                  key={entry.group}
                  label={entry.group}
                  defaultOpened
                  childrenOffset={16}
                >
                  {entry.itemList.map((item) =>
                    navLink(item.route, item.label),
                  )}
                </NavLink>
              ),
            )}
          </Stack>
        </AppShell.Section>
      </AppShell.Navbar>
      <AppShell.Main>
        {authState === 'YES' && apiUnreachable && <ApiUnreachableBanner />}
        <div className="app-page" ref={pageRef}>
          {authState === 'UNREACHABLE' ? <ApiUnreachableNotice /> : <Outlet />}
        </div>
      </AppShell.Main>
    </AppShell>
  )
}

function App() {
  return <RouterProvider router={router} />
}

export default App
