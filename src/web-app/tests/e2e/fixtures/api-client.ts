import { E2E_API_ROOT, E2E_BASE_URL } from "../../../playwright.config";

// A minimal API client for building fixtures.
//
// Fixtures are created through the HTTP API rather than INSERTed into the
// database, so every row a spec exercises has passed the same validation,
// grant checks and tenant scoping as production traffic. It also means the
// seed fails loudly when an endpoint breaks, which is a better signal than
// twenty specs failing on a missing element.

export interface ApiSession {
  cookie: string;
  csrfToken: string;
  securityContext: number;
}

interface Envelope<TData> {
  data: TData;
}

const request = async <TResult>(
  session: ApiSession | null,
  method: string,
  path: string,
  body?: unknown,
): Promise<TResult> => {
  const headers: Record<string, string> = { Origin: E2E_BASE_URL };
  if (session) {
    headers.Cookie = session.cookie;
    headers["X-CSRF-Token"] = session.csrfToken;
  }
  if (body !== undefined) headers["Content-Type"] = "application/json";

  const response = await fetch(E2E_API_ROOT + path, {
    method,
    headers,
    body: body === undefined ? undefined : JSON.stringify(body),
  });

  const text = await response.text();
  if (!response.ok) {
    throw new Error(
      `fixture ${method} ${path} failed: ${response.status} ${text}`,
    );
  }
  return text ? (JSON.parse(text) as TResult) : (undefined as TResult);
};

export const signIn = async (
  userName: string,
  password: string,
): Promise<ApiSession> => {
  const response = await fetch(`${E2E_API_ROOT}/auth`, {
    method: "POST",
    headers: { "Content-Type": "application/json", Origin: E2E_BASE_URL },
    body: JSON.stringify({ userName, password }),
  });
  const text = await response.text();
  if (!response.ok) {
    throw new Error(
      `fixture sign-in failed: ${response.status} ${text}. ` +
        `Has bin/513-reset-e2e-db been run?`,
    );
  }

  const parsed = JSON.parse(text) as {
    csrfToken: string;
    user: { securityContext: number };
  };
  const cookie = (response.headers.getSetCookie?.() ?? [])
    .map((value) => value.split(";")[0])
    .join("; ");
  if (!cookie) throw new Error("fixture sign-in returned no auth cookie");

  return {
    cookie,
    csrfToken: parsed.csrfToken,
    securityContext: parsed.user.securityContext,
  };
};

export const api = {
  get: <TData>(session: ApiSession, path: string) =>
    request<Envelope<TData>>(session, "GET", path).then((body) => body.data),
  post: <TData>(session: ApiSession, path: string, body?: unknown) =>
    request<Envelope<TData>>(session, "POST", path, body).then(
      (envelope) => envelope.data,
    ),
  put: <TData>(session: ApiSession, path: string, body: unknown) =>
    request<Envelope<TData>>(session, "PUT", path, body).then(
      (envelope) => envelope.data,
    ),
  delete: <TData>(session: ApiSession, path: string) =>
    request<Envelope<TData>>(session, "DELETE", path).then(
      (envelope) => envelope.data,
    ),
};

/** The auth cookie as a Playwright storageState cookie entry. */
export const cookieForStorageState = (session: ApiSession) => {
  const [name, ...rest] = session.cookie.split("=");
  return {
    name,
    value: rest.join("="),
    domain: "localhost",
    path: "/",
    expires: -1,
    httpOnly: true,
    secure: false,
    sameSite: "Strict" as const,
  };
};
