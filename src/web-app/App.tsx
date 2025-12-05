import { createBrowserRouter, RouterProvider, Outlet } from 'react-router-dom'
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
         path: 'example/:id',
         element: <ExampleEditPage />,
       },
    ],
  },
])

function AppLayout() {
  return <Outlet />
}

function App() {
  return <RouterProvider router={router} />
}

export default App
