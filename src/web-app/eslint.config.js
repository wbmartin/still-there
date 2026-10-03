import js from '@eslint/js'
import globals from 'globals'
import reactHooks from 'eslint-plugin-react-hooks'
import reactRefresh from 'eslint-plugin-react-refresh'
import tseslint from 'typescript-eslint'
import { defineConfig, globalIgnores } from 'eslint/config'

export default defineConfig([
  globalIgnores(['dist']),
  {
    files: ['**/*.{ts,tsx}'],
    extends: [
      js.configs.recommended,
      tseslint.configs.recommendedTypeChecked,
      reactHooks.configs.flat.recommended,
      reactRefresh.configs.vite,
    ],
    languageOptions: {
      ecmaVersion: 2020,
      globals: globals.browser,
      parserOptions: {
        project: ['./tsconfig.node.json', './tsconfig.app.json', './tsconfig.test.json'],
        tsconfigRootDir: import.meta.dirname,
      },
    },
    rules: {
      // Errors go through utils/report-error.ts, never straight to the console.
      'no-console': 'error',
      // The API is called through apiJson (utils/api.ts), never a bare fetch.
      'no-restricted-globals': [
        'error',
        {
          name: 'fetch',
          message: 'Call the API through apiJson (utils/api.ts).',
        },
      ],
    },
  },
  {
    // The one place app code may write to the console; the test harnesses
    // print their own progress.
    files: ['utils/report-error.ts', 'tests/**'],
    rules: { 'no-console': 'off' },
  },
  {
    // The one place app code may fetch; tests stub and call it directly.
    files: ['utils/api.ts', 'tests/**'],
    rules: { 'no-restricted-globals': 'off' },
  },
])
