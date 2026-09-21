import { beforeEach, describe, expect, it, vi } from "vitest";
import { ApiUnreachableError, apiRequest, isApiUnreachable } from "../utils/api";
import { useAuthStore } from "../stores/auth-store";

// The distinction these tests protect: "the server said no" and "the server did
// not answer" are different facts, and the app used to report both as "you are
// not signed in".
//
// It is worth stating why this needs a test rather than a careful reading.
// `fetch` only REJECTS on a connection-level failure; a 401 and a 500 both
// resolve normally. So the two cases arrive at a `catch` block by completely
// different routes, and a refactor that adds a `throw` to the success path
// would silently reclassify an ordinary HTTP failure as an outage. Neither
// mistake shows up in a typecheck, and both are invisible until somebody is
// staring at a login form during an incident.

const resetAuthStore = () => {
  useAuthStore.setState({
    authState: "TBD",
    grantList: [],
    availableSecurityContextList: [],
    userName: "",
    securityContext: 0,
    loading: false,
    error: null,
  });
};

describe("apiRequest network-failure classification", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    resetAuthStore();
  });

  it("wraps a connection failure in ApiUnreachableError", async () => {
    // What the browser actually throws when the server is not listening.
    vi.stubGlobal(
      "fetch",
      vi.fn().mockRejectedValue(new TypeError("Failed to fetch")),
    );

    await expect(apiRequest("/refresh", {})).rejects.toBeInstanceOf(
      ApiUnreachableError,
    );
  });

  it("preserves the original failure as the cause", async () => {
    const original = new TypeError("Failed to fetch");
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(original));

    await expect(apiRequest("/refresh", {})).rejects.toMatchObject({
      cause: original,
    });
  });

  it("does NOT classify an HTTP error response as unreachable", async () => {
    // The load-bearing half. A 401 resolves — it is a real answer from a
    // reachable server — so it must travel back to the caller as a normal
    // response and be judged on `ok`, not raised as an outage.
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({ ok: false, status: 401 }),
    );

    const response = await apiRequest("/refresh", {});

    expect(response.ok).toBe(false);
    expect(response.status).toBe(401);
  });

  it("rethrows an abort unchanged rather than calling it an outage", async () => {
    // Navigating away from a slow page cancels its requests. Reporting that as
    // a server failure would put an outage screen in front of somebody whose
    // server is perfectly healthy.
    const abort = new DOMException("The operation was aborted.", "AbortError");
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(abort));

    await expect(apiRequest("/refresh", {})).rejects.toBe(abort);
    expect(isApiUnreachable(abort)).toBe(false);
  });
});

describe("auth store: unreachable vs unauthenticated", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    resetAuthStore();
  });

  it("reports UNREACHABLE when the session check cannot reach the server", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockRejectedValue(new TypeError("Failed to fetch")),
    );

    await useAuthStore.getState().authenticateWithAuthToken();

    // NOT "NO". Answering "NO" here is the bug: it means the app claims to
    // know the person is signed out when it has learned nothing at all, and
    // sends them to a login form that cannot work either.
    expect(useAuthStore.getState().authState).toBe("UNREACHABLE");
  });

  it("reports NO when the server answers that there is no session", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({ ok: false, status: 401 }),
    );

    await useAuthStore.getState().authenticateWithAuthToken();

    expect(useAuthStore.getState().authState).toBe("NO");
  });

  // The cases a `response.ok` check gets wrong, and the reason this is a
  // status question rather than a network question.
  //
  // In production nginx answers 502 while the Go server restarts, and Vite's
  // dev proxy behaves the same way. Those are ordinary HTTP responses — no
  // network error is thrown — so classifying purely on the fetch rejection
  // would send everyone straight back to the login screen the moment the API
  // bounced. Nobody asked the server about the session; it never got the
  // question.
  it.each([500, 502, 503, 504])(
    "reports UNREACHABLE when a gateway or server error answers with %i",
    async (status) => {
      vi.stubGlobal(
        "fetch",
        vi.fn().mockResolvedValue({ ok: false, status }),
      );

      await useAuthStore.getState().authenticateWithAuthToken();

      expect(useAuthStore.getState().authState).toBe("UNREACHABLE");
    },
  );

  it("reports NO on a 403, which IS the server refusing this person", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({ ok: false, status: 403 }),
    );

    await useAuthStore.getState().authenticateWithAuthToken();

    expect(useAuthStore.getState().authState).toBe("NO");
  });

  it("does not claim a session ended because the body would not parse", async () => {
    // An HTML error page from a proxy arriving with a 200 is the shape here.
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        json: () => Promise.reject(new SyntaxError("Unexpected token <")),
      }),
    );

    await useAuthStore.getState().authenticateWithAuthToken();

    expect(useAuthStore.getState().authState).toBe("UNREACHABLE");
  });

  it("reports YES on a successful session check", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: () =>
          Promise.resolve({
            user: {
              userName: "someone@example.com",
              grantList: [1050],
              availableSecurityContextList: [1],
              securityContext: 1,
            },
            csrfToken: "token",
          }),
      }),
    );

    await useAuthStore.getState().authenticateWithAuthToken();

    expect(useAuthStore.getState().authState).toBe("YES");
  });

  it("keeps a failed SIGN-IN on the login form, with an accurate message", async () => {
    // Deliberately still "NO" rather than "UNREACHABLE": the person is standing
    // at the form, and replacing it with an outage screen would take away the
    // thing they need to retry. The message is what has to carry the truth.
    vi.stubGlobal(
      "fetch",
      vi.fn().mockRejectedValue(new TypeError("Failed to fetch")),
    );

    await useAuthStore
      .getState()
      .authenticateWithUsernamePassword("someone@example.com", "pw");

    const state = useAuthStore.getState();
    expect(state.authState).toBe("NO");
    expect(state.error).toMatch(/could not reach the server/i);
  });

  it("does not blame the connection for a rejected password", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({ ok: false, status: 401 }),
    );

    await useAuthStore
      .getState()
      .authenticateWithUsernamePassword("someone@example.com", "wrong");

    const state = useAuthStore.getState();
    expect(state.authState).toBe("NO");
    expect(state.error).not.toMatch(/could not reach the server/i);
  });
});
