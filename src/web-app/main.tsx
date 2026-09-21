import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { MantineProvider } from '@mantine/core'
import { ModalsProvider } from '@mantine/modals'
import { Notifications } from '@mantine/notifications'
import '@mantine/core/styles.css'
import './index.css'
import App from './app.tsx'
import ErrorBoundary from './components/error-boundary.tsx'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <MantineProvider>
      <ModalsProvider>
        <Notifications position="bottom-right" />
        <ErrorBoundary>
          <App />
        </ErrorBoundary>
      </ModalsProvider>
    </MantineProvider>
  </StrictMode>,
)
