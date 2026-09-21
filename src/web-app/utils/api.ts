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
 * The request never reached the server.
 *
 * A 401 or a 500 RESOLVES fetch with a response; only a connection-level
 * failure (refused, DNS, CORS, offline) rejects, and it rejects with a bare
 * `TypeError: Failed to fetch`. Naming it lets callers tell an outage the user
 * can wait out from a refusal that signing in again would fix.
 */
export class ApiUnreachableError extends Error {
  constructor(url: string, options?: { cause?: unknown }) {
    super(`Could not reach the server (${url})`);
    this.name = "ApiUnreachableError";
    if (options?.cause !== undefined) this.cause = options.cause;
  }
}

export const isApiUnreachable = (error: unknown): boolean =>
  error instanceof ApiUnreachableError;

const apiRequest = async (
  url: string,
  options: RequestInit = {},
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
    throw new ApiUnreachableError(url, { cause });
  }
};

/**
 * The server said this person is not signed in.
 *
 * 403 is included: CSRF validation rejects with 403, and from the user's side
 * that is the same event -- the credential they were carrying is no longer
 * good.
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

// Reported through a callback rather than by importing the auth store, which
// imports this module -- a direct call would be a cycle. The store registers
// itself once at startup.
let sessionExpiredListener: (() => void) | null = null;

export const setSessionExpiredListener = (
  listener: (() => void) | null,
): void => {
  sessionExpiredListener = listener;
};

/**
 * Turn a failed response into the right error, and tell the app when the
 * reason was the session. Call this instead of throwing on !response.ok.
 */
export const throwForFailedResponse = async (
  response: Response,
  readMessage: (response: Response) => Promise<string>,
): Promise<never> => {
  const message = await readMessage(response);
  if (response.status === 401 || response.status === 403) {
    sessionExpiredListener?.();
    throw new SessionExpiredError(response.status, message);
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
 * response.ok. By default a 401/403 is a SessionExpiredError and the app is
 * told; `anonymous: true` is for the endpoints reached before there is a
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
