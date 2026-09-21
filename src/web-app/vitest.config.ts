import react from "@vitejs/plugin-react";
import { defineConfig } from "vitest/config";

// The react plugin is what lets a test render a component; without it JSX in
// a .tsx test is never transformed.
export default defineConfig({
  plugins: [react()],
  test: {
    environment: "jsdom",
    globals: true,
    include: ["tests/**/*.test.ts", "tests/**/*.test.tsx"],
    coverage: {
      provider: "v8",
      include: ["stores/**", "utils/**", "hooks/**", "constants/**"],
      // A floor, not a target: the logic that is testable without a page.
      thresholds: { lines: 40, functions: 40 },
    },
  },
});
