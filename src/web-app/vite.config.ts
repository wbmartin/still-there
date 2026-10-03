import { defineConfig, loadEnv } from "vite";
import react from "@vitejs/plugin-react";

// https://vite.dev/config/
export default defineConfig(({ command, mode }) => {
  // Load environment variables for the current mode
  // (development, production, etc.)
  const env = loadEnv(mode, process.cwd(), '');

  // The API the dev server proxies /api to. Comes from .env.development
  // (or the process environment -- playwright.config.ts sets it for the e2e
  // stack); there is no fallback, so a missing value fails here, by name.
  // Only the dev server proxies: `vite build` never uses it (nginx serves
  // /api in production), so a build does not require it.
  const apiProxyUrl = env.VITE_API_PROXY_URL;
  if (command === "serve" && !apiProxyUrl) {
    throw new Error("VITE_API_PROXY_URL is not set (src/web-app/.env.development)");
  }

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
