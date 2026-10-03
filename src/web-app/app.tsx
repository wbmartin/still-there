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
  IconSettings,
  IconUserCircle,
} from '@tabler/icons-react'
import { useEffect, useRef } from 'react'
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

import { useAuthStore, setRouteNavigator } from './stores/auth-store'
import { ROUTES } from './constants/routes'
import { grants } from './constants/grants'
import { APP_NAME } from './constants/app'

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
             path: ROUTES.USER_PREFERENCES,
             element: <UserPreferencesPage />,
             errorElement: <ErrorPage />,
           },
    ],
  },
])

setRouteNavigator((path) => void router.navigate(path))

const HEADER_HEIGHT = 56;

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
  const { colorScheme } = usePreference();
  const { setColorScheme } = useMantineColorScheme();
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

  useEffect(() => {
    setColorScheme(colorScheme);
  }, [colorScheme, setColorScheme]);

  const handleLogout = async () => {
    closeNav();
    await logout();
    void navigate(ROUTES.LOGIN);
  };

  // Authorization checks for navigation
  const canManageUsers = grantList.includes(grants.CREATE_USER);
  const canManageUserGroups = grantList.includes(grants.CREATE_USER_GROUP);

  // Closing on tap (not on route change) also closes the phone drawer when
  // the tapped link is the page already shown.
  const navLink = (to: string, label: string) => (
    <NavLink component={Link} to={to} label={label} onClick={closeNav} />
  );

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
      </AppShell.Header>
      <AppShell.Navbar p="xs">
        <AppShell.Section grow component={ScrollArea} type="auto">
          <Stack gap={2}>
            {navLink(ROUTES.DASHBOARD, 'Dashboard')}
            {navLink(ROUTES.EXAMPLE_LIST, 'Examples')}
            {canManageUsers && navLink(ROUTES.APP_USER_LIST, 'User Admin')}
            {canManageUserGroups &&
              navLink(ROUTES.APP_USER_GROUP_LIST, 'User Groups')}
            {navLink(ROUTES.USER_PREFERENCES, 'Profile')}
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
