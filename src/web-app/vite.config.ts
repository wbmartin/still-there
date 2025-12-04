import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  root: '.',
  plugins: [react()],
  server: {
    proxy: {
      '/api': {
        target: 'https://localhost:8443', // Replace with the actual API URL from etc/loc-api-srv.env
        changeOrigin: true,
        secure: false, // Allow self-signed certs in dev
        rewrite: (path) => path.replace(/^\/api/, ''), // Remove /api prefix when forwarding
      },
    },
  },
})
