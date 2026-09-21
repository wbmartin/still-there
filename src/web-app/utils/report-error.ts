/**
 * The one place errors are written out. Everything else calls this instead
 * of console.error (ESLint no-console enforces it), so where errors go --
 * today the browser console, tomorrow maybe POST /error-report -- is one
 * decision, not twenty-six.
 *
 * context is a short noun phrase for what was being done: "deleting example",
 * "loading user list". The error is passed through untouched so the console
 * keeps its stack.
 */
export const reportError = (error: unknown, context: string): void => {
  console.error(`[${context}]`, error);
};
