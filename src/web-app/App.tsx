import { createBrowserRouter, RouterProvider, Outlet, Link } from 'react-router-dom'
import Login from './pages/login'
import Dashboard from './pages/dashboard'
import ExampleListPage from './pages/example-list-page'
import ExampleEditPage from './pages/example-edit-page'

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
  return (
    <div style={{ display: 'flex' }}>
      <nav style={{ width: '200px', padding: '1rem', borderRight: '1px solid #ccc' }}>
        <ul style={{ listStyle: 'none', padding: 0 }}>
          <li><Link to="/dashboard">Dashboard</Link></li>
          <li><Link to="/exampleList">Examples</Link></li>
        </ul>
      </nav>
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
