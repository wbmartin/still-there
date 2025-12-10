import { createBrowserRouter, RouterProvider, Outlet, Link, useNavigate } from 'react-router-dom'
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
  const { logout, authState } = useAuthStore();
  const navigate = useNavigate();

  const handleLogout = async () => {
    await logout();
    navigate(`/${ROUTES.LOGIN}`);
  };

  return (
    <div style={{ display: 'flex' }}>
      {authState === 'YES' && (
        <nav style={{ width: '200px', padding: '1rem', borderRight: '1px solid #ccc' }}>
          <ul style={{ listStyle: 'none', padding: 0 }}>
             <li><Link to={`/${ROUTES.DASHBOARD}`}>Dashboard</Link></li>
             <li><Link to={`/${ROUTES.EXAMPLE_LIST}`}>Examples</Link></li>
             <li><Link to={`/${ROUTES.APP_USER_LIST}`}>User Admin</Link></li>
             <li><Link to={`/${ROUTES.USER_GROUP_LIST}`}>User Groups</Link></li>
            <li><button onClick={handleLogout} style={{ background: 'none', border: 'none', color: 'blue', cursor: 'pointer', textDecoration: 'underline' }}>Logout</button></li>
          </ul>
        </nav>
      )}
      <main style={{ flex: 1, padding: '1rem' }}>
        <Outlet />
      </main>
    </div>
  )
}

function App() {
  return <RouterProvider router={router} />
}

export default App
