import { API_ENDPOINTS } from "../constants/api";
import type { RefreshResponse } from "../types/auth-type";

let csrfToken: string | null = null;

export const getCsrfToken = (): string | null => {
  return csrfToken;
};

export const setCsrfToken = (token: string): void => {
  csrfToken = token;
};

export const clearCsrfToken = (): void => {
  csrfToken = null;
};

/**
 * The API did not answer, or said "not now": a rejected fetch (refused, DNS,
 * CORS, offline), a 5xx (nginx's 502 while the API restarts, a 503) or a 429
 * (the per-minute rate limit, an account lockout).
 *
 * A 401 or a 500 RESOLVES fetch with a response; only a connection-level
 * failure rejects, with a bare `TypeError: Failed to fetch`. apiJson
 * folds both shapes into this one error so callers can tell an outage the
 * user can wait out from a refusal that signing in again would fix. It says
 * nothing about the session.
 */
export class ApiUnreachableError extends Error {
  /** The status that made it unreachable; undefined when fetch failed. */
  readonly status?: number;
  constructor(
    message: string,
    options: { status?: number; cause?: unknown } = {},
  ) {
    super(message);
    this.name = "ApiUnreachableError";
    this.status = options.status;
    if (options.cause !== undefined) this.cause = options.cause;
  }
}

export const isApiUnreachable = (
  error: unknown,
): error is ApiUnreachableError => error instanceof ApiUnreachableError;

/**
 * What a call said about the API and the session. Two subscribers:
 * stores/api-status-store.ts (the "server not answering" banner, raised by
 * `unreachable`, cleared by the next `ok`) and stores/auth-store.ts (sign
 * out on `session-expired`, and nothing else).
 */
export type ApiOutcome =
  | { kind: "ok" }
  | { kind: "unreachable"; error: ApiUnreachableError }
  | { kind: "session-expired"; error: SessionExpiredError };

type ApiOutcomeListener = (outcome: ApiOutcome) => void;

const apiOutcomeListenerList: ApiOutcomeListener[] = [];

/** Listen to every call's outcome; returns the unsubscribe. */
export const subscribeApiOutcome = (
  listener: ApiOutcomeListener,
): (() => void) => {
  apiOutcomeListenerList.push(listener);
  return () => {
    const at = apiOutcomeListenerList.indexOf(listener);
    if (at >= 0) apiOutcomeListenerList.splice(at, 1);
  };
};

const announce = (outcome: ApiOutcome): void => {
  // A copy: a listener may unsubscribe itself while being called.
  for (const listener of [...apiOutcomeListenerList]) listener(outcome);
};

// The fetch itself, with the CSRF header and the outage mapping. An
// anonymous call (no session yet, or one that is ending) sends no CSRF
// header, so a token left over from an ended session does not ride along.
const sendRequest = async (
  url: string,
  options: RequestInit,
  anonymous = false,
) => {
  const headers = new Headers(options.headers);
  const token = getCsrfToken();
  if (!anonymous && token) {
    headers.set("X-CSRF-Token", token);
  }
  try {
    return await fetch(`/api${url}`, {
      ...options,
      headers,
      credentials: "include",
    });
  } catch (cause) {
    // An abort is a deliberate cancellation by our own code, not an outage.
    if (cause instanceof DOMException && cause.name === "AbortError") {
      throw cause;
    }
    throw new ApiUnreachableError(`Could not reach the server (${url})`, {
      cause,
    });
  }
};

/**
 * The write was dropped because the session cookie now belongs to another
 * user or tenant (signed in or switched in another tab). The tab has already
 * moved to that identity; a caller can ignore this error.
 */
export class SessionChangedError extends Error {
  constructor() {
    super("Your session changed in another tab; this change was not saved.");
    this.name = "SessionChangedError";
  }
}

export const isSessionChanged = (error: unknown): boolean =>
  error instanceof SessionChangedError;

/**
 * How the API layer asks the auth store about a refreshed session, through
 * a callback: the auth store imports this module, so a direct call would
 * be a cycle.
 */
export interface SessionResyncListener {
  /** The refreshed session is the user and tenant this tab shows. */
  isSameIdentity: (refreshed: RefreshResponse) => boolean;
  /** It is not: the tab moves to the refreshed session. */
  adoptSession: (refreshed: RefreshResponse) => void;
}

let sessionResyncListener: SessionResyncListener | null = null;

export const setSessionResyncListener = (
  listener: SessionResyncListener | null,
): void => {
  sessionResyncListener = listener;
};

type ResyncOutcome = "SAME" | "CHANGED" | "FAILED";

const SAFE_METHOD_LIST = ["GET", "HEAD", "OPTIONS", "TRACE"];

// A write the server refused for its CSRF token: another tab's sign-in or
// context switch moved the shared cookie to a session this tab's token does
// not match.
const isCsrfRefusal = async (
  response: Response,
  options: RequestInit,
): Promise<boolean> => {
  if (response.status !== 403) return false;
  const method = (options.method || "GET").toUpperCase();
  if (SAFE_METHOD_LIST.includes(method)) return false;
  try {
    const body = (await response.clone().json()) as { code?: string };
    return body.code === "CSRF";
  } catch {
    return false;
  }
};

const refreshSession = async (): Promise<ResyncOutcome> => {
  const listener = sessionResyncListener;
  if (!listener) return "FAILED";
  try {
    const response = await sendRequest(API_ENDPOINTS.REFRESH, {});
    if (!response.ok) return "FAILED";
    const refreshed = (await response.json()) as RefreshResponse;
    if (!refreshed.user || !refreshed.csrfToken) return "FAILED";
    if (!listener.isSameIdentity(refreshed)) {
      listener.adoptSession(refreshed);
      return "CHANGED";
    }
    setCsrfToken(refreshed.csrfToken);
    return "SAME";
  } catch {
    return "FAILED";
  }
};

// Single flight: writes refused together share one /refresh and one verdict,
// so a session is adopted once and no write is resent under it.
let pendingResync: Promise<ResyncOutcome> | null = null;

const resyncSession = (): Promise<ResyncOutcome> => {
  pendingResync ??= refreshSession().finally(() => {
    pendingResync = null;
  });
  return pendingResync;
};

// A write refused for a stale CSRF token is resent once after /refresh
// supplies the current token, if the session is still the user and tenant
// this tab shows; if it is not, the write is dropped and the tab moves to the
// new identity (SessionChangedError). A second refusal, or a failed refresh,
// returns the 403 as it came.
const sendWithCsrfResync = async (url: string, options: RequestInit) => {
  const response = await sendRequest(url, options);
  if (!(await isCsrfRefusal(response, options))) return response;
  const outcome = await resyncSession();
  if (outcome === "CHANGED") throw new SessionChangedError();
  if (outcome === "FAILED") return response;
  return sendRequest(url, options);
};

// 5xx and 429 mean "the API is there but cannot answer this now": wait and
// try again, the same advice as no answer at all.
const isUnavailableStatus = (status: number): boolean =>
  status >= 500 || status === 429;

/**
 * fetch against the API, with the CSRF resync above (a session call only;
 * an anonymous one sends no token to be refused). A 5xx or 429 throws
 * ApiUnreachableError carrying the server's message, as a failed fetch does.
 * Every call announces its outcome: `ok` on a 2xx, `unreachable` on an
 * ApiUnreachableError. Other statuses return to apiJson as they came.
 *
 * Private on purpose: a second, raw path is how `response.ok` handling grew
 * in ten places. Everything goes through apiJson.
 */
const apiRequest = async (
  url: string,
  options: RequestInit,
  anonymous: boolean,
) => {
  try {
    const response = anonymous
      ? await sendRequest(url, options, true)
      : await sendWithCsrfResync(url, options);
    if (isUnavailableStatus(response.status)) {
      throw new ApiUnreachableError(await readErrorMessage(response), {
        status: response.status,
      });
    }
    if (response.ok) announce({ kind: "ok" });
    return response;
  } catch (error) {
    if (isApiUnreachable(error)) announce({ kind: "unreachable", error });
    throw error;
  }
};

/**
 * The API answered and refused: any non-2xx that is not an outage. It
 * carries the status so a caller can act on one it expects (a 404 read as
 * "no such row", a 409, /refresh's 401/403 as "signed out") and the server's
 * `{"error"}` text as its message.
 */
export class ApiError extends Error {
  readonly status: number;
  constructor(status: number, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

/** The call was refused with this status (never an outage). */
export const isApiStatus = (error: unknown, ...statusList: number[]) =>
  error instanceof ApiError && statusList.includes(error.status);

/**
 * The server said this person is not signed in: a 401, which CheckJWT
 * writes for a missing, invalid, expired or revoked `auth` cookie.
 *
 * A 403 is not this. It is a refusal -- a missing grant, a blocked account,
 * a CSRF token apiRequest could not resync, a cross-tenant refusal -- and
 * signing in again fixes none of them. It stays a plain ApiError carrying
 * the server's message, and the session is untouched.
 */
export class SessionExpiredError extends ApiError {
  constructor(status: number, message?: string) {
    super(status, message || "Your session has ended. Please sign in again.");
    this.name = "SessionExpiredError";
  }
}

export const isSessionExpired = (error: unknown): boolean =>
  error instanceof SessionExpiredError;

// The message a failed response carries, or a generic one.
const readErrorMessage = async (response: Response): Promise<string> => {
  try {
    const body = (await response.json()) as {
      error?: string;
      message?: string;
    };
    return body.error || body.message || `Request failed (${response.status})`;
  } catch {
    return `Request failed (${response.status})`;
  }
};

/**
 * The one way to call the API. A 2xx is parsed as JSON (an empty body, a
 * 204 or a bare 200, is undefined); anything else throws, so no caller looks
 * at response.ok:
 *   - a failed fetch, 5xx or 429: ApiUnreachableError;
 *   - a 401: SessionExpiredError, and the app is told (it signs out);
 *   - any other status: ApiError with that status and the server's message.
 * `anonymous: true` is for the endpoints reached without a session (login,
 * MFA setup/verify, invites, self-registration) or leaving one (/logoff): no
 * CSRF header is sent, and a 401 means "wrong password/code" or "already
 * gone", an ApiError that must not touch the session.
 */
export const apiJson = async <T>(
  url: string,
  options: RequestInit = {},
  { anonymous = false }: { anonymous?: boolean } = {},
): Promise<T> => {
  const response = await apiRequest(url, options, anonymous);
  if (!response.ok) {
    const message = await readErrorMessage(response);
    if (response.status === 401 && !anonymous) {
      const error = new SessionExpiredError(response.status, message);
      announce({ kind: "session-expired", error });
      throw error;
    }
    throw new ApiError(response.status, message);
  }
  // Always read the body: one left unread keeps the request open.
  const text = await response.text();
  return (text ? JSON.parse(text) : undefined) as T;
};

/** JSON body + header for the POST/PUT calls. */
export const jsonBody = (body: unknown): RequestInit => ({
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify(body),
});
