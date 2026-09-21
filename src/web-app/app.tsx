import {
  createBrowserRouter,
  RouterProvider,
  Outlet,
  Link,
  useNavigate,
} from 'react-router-dom'
import {
  NavLink,
  Button,
  Stack,
  Center,
  Container,
  Flex,
  Alert,
  Text,
} from '@mantine/core'
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

import { useAuthStore } from './stores/auth-store'
import { ROUTES } from './constants/routes'
import { grants } from './constants/grants'

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

function AppLayout() {
  const { logout, authState, grantList } = useAuthStore();
  const navigate = useNavigate();

  const handleLogout = async () => {
    await logout();
    void navigate(`/${ROUTES.LOGIN}`);
  };

  // Authorization checks for navigation
  const canManageUsers = grantList.includes(grants.CREATE_USER);
  const canManageUserGroups = grantList.includes(grants.CREATE_USER_GROUP);

  return (
    <Center w="100%">
      <Container size="xl" w="100%">
        <Flex>
          {authState === 'YES' && (
            <nav style={{ width: '200px', padding: '1rem', borderRight: '1px solid var(--mantine-color-default-border)' }}>
              <Stack gap="xs">
                <NavLink component={Link} to={`/${ROUTES.DASHBOARD}`} label="Dashboard" />
                <NavLink component={Link} to={`/${ROUTES.EXAMPLE_LIST}`} label="Examples" />
                {canManageUsers && <NavLink component={Link} to={`/${ROUTES.APP_USER_LIST}`} label="User Admin" />}
                {canManageUserGroups && <NavLink component={Link} to={`/${ROUTES.APP_USER_GROUP_LIST}`} label="User Groups" />}
                <NavLink component={Link} to={`/${ROUTES.USER_PREFERENCES}`} label="Preferences" />
                <Button variant="subtle" onClick={() => void handleLogout()} fullWidth justify="flex-start">
                  Logout
                </Button>
              </Stack>
            </nav>
          )}
          <main style={{ flex: 1, padding: '1rem' }}>
            {authState === 'UNREACHABLE' ? <ApiUnreachableNotice /> : <Outlet />}
          </main>
        </Flex>
      </Container>
    </Center>
  )
}

function App() {
  return <RouterProvider router={router} />
}

export default App
