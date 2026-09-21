import { defineConfig, loadEnv } from "vite";
import react from "@vitejs/plugin-react";

// https://vite.dev/config/
export default defineConfig(({ mode }) => {
  // Load environment variables for the current mode
  // (development, production, etc.)
  const env = loadEnv(mode, process.cwd(), '');

  // Get API proxy URL from environment variable,
  // fallback to localhost for development
  // Set VITE_API_PROXY_URL in .env.development or .env.production files
  const apiProxyUrl = env.VITE_API_PROXY_URL || 'http://localhost:8080';

  return {
    root: ".",
    plugins: [react()],
    build: {
      chunkSizeWarningLimit: 3145728, // 3MB
    },
    server: {
      proxy: {
        "/api": {
          target: apiProxyUrl,
          changeOrigin: true,
          secure: false, // Allow self-signed certs in dev
          rewrite: (path) => path.replace(/^\/api/, ""), // Remove /api prefix when forwarding
        },
      },
    },
  };
});
