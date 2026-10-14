/// <reference types="vite/client" />

// Stamped in at build time by vite.config.ts (`define`): which build a tab is
// running. Issue reports carry them (stores/issue-report-store.ts).
interface ImportMetaEnv {
  readonly VITE_APP_VERSION?: string;
  readonly VITE_APP_BUILT_AT?: string;
}
