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
 * failure rejects, with a bare `TypeError: Failed to fetch`. apiRequest
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

// The fetch itself, with the CSRF header and the outage mapping.
const sendRequest = async (
  url: string,
  options: RequestInit,
  csrfToken?: string,
) => {
  const headers = new Headers(options.headers);
  const token = csrfToken || getCsrfToken();
  if (token) {
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
const sendWithCsrfResync = async (
  url: string,
  options: RequestInit,
  csrfToken?: string,
) => {
  const response = await sendRequest(url, options, csrfToken);
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
 * fetch against the API, with the CSRF resync above. A 5xx or 429 throws
 * ApiUnreachableError carrying the server's message, as a failed fetch does.
 * Every call announces its outcome: `ok` on a 2xx, `unreachable` on an
 * ApiUnreachableError. Other statuses return to the caller as they came.
 */
const apiRequest = async (
  url: string,
  options: RequestInit = {},
  csrfToken?: string,
) => {
  try {
    const response = await sendWithCsrfResync(url, options, csrfToken);
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
 * The server said this person is not signed in: a 401, which CheckJWT
 * writes for a missing, invalid, expired or revoked `auth` cookie.
 *
 * A 403 is not this. It is a refusal -- a missing grant, a blocked account,
 * a CSRF token apiRequest could not resync, a cross-tenant refusal -- and
 * signing in again fixes none of them. It stays a plain Error carrying the
 * server's message, and the session is untouched.
 */
export class SessionExpiredError extends Error {
  readonly status: number;
  constructor(status: number, message?: string) {
    super(message || "Your session has ended. Please sign in again.");
    this.name = "SessionExpiredError";
    this.status = status;
  }
}

export const isSessionExpired = (error: unknown): boolean =>
  error instanceof SessionExpiredError;

/**
 * Turn a failed response into the right error, and announce
 * `session-expired` on a 401 (the auth store signs out). Call this instead
 * of throwing on !response.ok.
 */
export const throwForFailedResponse = async (
  response: Response,
  readMessage: (response: Response) => Promise<string>,
): Promise<never> => {
  const message = await readMessage(response);
  if (response.status === 401) {
    const error = new SessionExpiredError(response.status, message);
    announce({ kind: "session-expired", error });
    throw error;
  }
  throw new Error(message);
};

/**
 * The message a failed response carries, or a generic one. Both the entity
 * stores and the one-off calls pass this to throwForFailedResponse.
 */
export const readErrorMessage = async (response: Response): Promise<string> => {
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
 * apiRequest + the failure handling every caller used to repeat: a non-2xx
 * response throws and a 2xx one is parsed as JSON, so callers never look at
 * response.ok. A 5xx or 429 is an ApiUnreachableError (from apiRequest).
 * By default a 401 is a SessionExpiredError and the app is told; `anonymous: true` is for the endpoints reached before there is a
 * session (login, MFA setup/verify, self-registration), where a 401 means
 * "wrong password/code", not "signed out", and must not touch the session.
 */
export const apiJson = async <T>(
  url: string,
  options: RequestInit = {},
  { anonymous = false }: { anonymous?: boolean } = {},
): Promise<T> => {
  const response = await apiRequest(url, options);
  if (!response.ok) {
    if (anonymous) throw new Error(await readErrorMessage(response));
    await throwForFailedResponse(response, readErrorMessage);
  }
  if (response.status === 204) return undefined as T;
  return (await response.json()) as T;
};

/** JSON body + header for the POST/PUT calls. */
export const jsonBody = (body: unknown): RequestInit => ({
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify(body),
});

export { apiRequest };
