import { createBrowserRouter, RouterProvider, Outlet, Link, useNavigate } from 'react-router-dom'
import Login from './pages/login'
import Dashboard from './pages/dashboard'
import ExampleListPage from './pages/example-list-page'
import ExampleEditPage from './pages/example-edit-page'
import { useAuthStore } from './stores/auth-store'

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
        path: 'login',
        element: <Login />,
      },
      {
        path: 'dashboard',
        element: <Dashboard />,
      },
        {
          path: 'exampleList',
          element: <ExampleListPage />,
        },
        {
           path: 'example',
           element: <ExampleEditPage />,
         },
    ],
  },
])

function AppLayout() {
  const { logout, authState } = useAuthStore();
  const navigate = useNavigate();

  const handleLogout = () => {
    logout();
    navigate('/login');
  };

  return (
    <div style={{ display: 'flex' }}>
      {authState === 'YES' && (
        <nav style={{ width: '200px', padding: '1rem', borderRight: '1px solid #ccc' }}>
          <ul style={{ listStyle: 'none', padding: 0 }}>
            <li><Link to="/dashboard">Dashboard</Link></li>
            <li><Link to="/exampleList">Examples</Link></li>
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
