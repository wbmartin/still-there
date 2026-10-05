/**
 * The one place errors are written out. Everything else calls this instead
 * of console.error (ESLint no-console enforces it), so where errors go is one
 * decision, not twenty-six. Today: the browser console only. The API's
 * POST /error-report needs a session, so a crash on the login screen could
 * never reach it; wiring it here would be its own ticket.
 *
 * context is a short noun phrase for what was being done: "deleting example",
 * "loading user list". The error is passed through untouched so the console
 * keeps its stack.
 */
export const reportError = (error: unknown, context: string): void => {
  console.error(`[${context}]`, error);
};
