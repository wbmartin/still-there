import { execSync } from "node:child_process";
import { defineConfig, loadEnv } from "vite";
import react from "@vitejs/plugin-react";

// Which build is running in a browser tab: the commit, stamped in at build
// time. The host build passes VITE_APP_VERSION (its GIT_SHA); otherwise the
// checkout's own commit; otherwise "unknown". Issue reports carry it, so a
// report from a tab opened before the last deploy is easy to spot.
const appVersion = (env: Record<string, string>): string => {
  if (env.VITE_APP_VERSION) return env.VITE_APP_VERSION;
  try {
    return execSync("git rev-parse --short=12 HEAD", {
      stdio: ["ignore", "pipe", "ignore"],
    })
      .toString()
      .trim();
  } catch {
    return "unknown";
  }
};

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
    define: {
      "import.meta.env.VITE_APP_VERSION": JSON.stringify(appVersion(env)),
      "import.meta.env.VITE_APP_BUILT_AT": JSON.stringify(
        new Date().toISOString(),
      ),
    },
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
