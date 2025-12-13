import { createBrowserRouter, RouterProvider, Outlet, Link, useNavigate } from 'react-router-dom'
import { NavLink, Button, Stack } from '@mantine/core'
import Login from './pages/login'
import Dashboard from './pages/dashboard'
import ExampleListPage from './pages/example-list-page'
import ExampleEditPage from './pages/example-edit-page'
import AppUserListPage from './pages/app-user-list-page'
import AppUserEditPage from './pages/app-user-edit-page'
import UserGroupListPage from './pages/user-group-list-page'
import UserGroupEditPage from './pages/user-group-edit-page'

import { useAuthStore } from './stores/auth-store'
import { ROUTES } from './constants/routes'
import { grants } from './constants/grants'

const router = createBrowserRouter([
  {
    path: '/',
    element: <AppLayout />,
    children: [
      {
        index: true,
        element: <Login />,
      },
       {
         path: ROUTES.LOGIN,
         element: <Login />,
       },
       {
         path: ROUTES.DASHBOARD,
         element: <Dashboard />,
       },
       {
         path: ROUTES.EXAMPLE_LIST,
         element: <ExampleListPage />,
       },
         {
           path: 'example',
           element: <ExampleEditPage />,
         },
         {
           path: ROUTES.APP_USER_LIST,
           element: <AppUserListPage />,
         },
         {
           path: ROUTES.APP_USER_EDIT,
           element: <AppUserEditPage />,
         },
         {
           path: ROUTES.USER_GROUP_LIST,
           element: <UserGroupListPage />,
         },
         {
           path: ROUTES.USER_GROUP_EDIT,
           element: <UserGroupEditPage />,
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
    <div style={{ display: 'flex', justifyContent: 'center', width: '100%' }}>
      <div style={{ display: 'flex', maxWidth: '1024px', width: '100%' }}>
        {authState === 'YES' && (
        <nav style={{ width: '200px', padding: '1rem', borderRight: '1px solid #ccc' }}>
          <Stack gap="xs">
            <NavLink component={Link} to={`/${ROUTES.DASHBOARD}`} label="Dashboard" />
            <NavLink component={Link} to={`/${ROUTES.EXAMPLE_LIST}`} label="Examples" />
            {canManageUsers && <NavLink component={Link} to={`/${ROUTES.APP_USER_LIST}`} label="User Admin" />}
            {canManageUserGroups && <NavLink component={Link} to={`/${ROUTES.USER_GROUP_LIST}`} label="User Groups" />}
             <Button variant="subtle" onClick={() => void handleLogout()} fullWidth style={{ justifyContent: 'flex-start' }}>
               Logout
             </Button>
           </Stack>
        </nav>
        )}
        <main style={{ flex: 1, padding: '1rem' }}>
          <Outlet />
        </main>
      </div>
    </div>
  )
}

function App() {
  return <RouterProvider router={router} />
}

export default App
