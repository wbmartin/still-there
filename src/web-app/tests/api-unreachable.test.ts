import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { notifications } from "@mantine/notifications";
import {
  ApiUnreachableError,
  SessionExpiredError,
  apiJson,
  isApiUnreachable,
  subscribeApiOutcome,
  type ApiOutcome,
} from "../utils/api";
import { useAuthStore } from "../stores/auth-store";
import { useApiStatusStore } from "../stores/api-status-store";

// The distinction these tests protect: "the server said no" and "the server
// did not answer" are different facts, and the app used to report both as
// "you are not signed in".
//
// It is worth stating why this needs a test rather than a careful reading.
// `fetch` only REJECTS on a connection-level failure; a 401 and a 502 both
// resolve normally. apiJson folds a rejected fetch, a 5xx and a 429 into
// one ApiUnreachableError, and a refactor that moves one status across that
// line reclassifies an outage as a sign-out (or the reverse). Neither shows
// up in a typecheck.

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

const json = (status: number, body: unknown = {}) =>
  new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });

const refreshAnswer = () =>
  json(200, {
    user: {
      userName: "someone@example.com",
      grantList: [1050],
      availableSecurityContextList: [2],
      securityContext: 2,
    },
    csrfToken: "token",
  });

// fetch answering each call with the next factory in the list.
const stubFetchSequence = (answerList: (() => Promise<Response>)[]) => {
  const fetchMock = vi.fn(() => {
    const answer = answerList.shift();
    if (!answer) throw new Error("unexpected fetch");
    return answer();
  });
  vi.stubGlobal("fetch", fetchMock);
  return fetchMock;
};

const answerWith = (status: number) => () => Promise.resolve(json(status));
const failedFetch = () => Promise.reject(new TypeError("Failed to fetch"));

const recordOutcomes = () => {
  const outcomeList: ApiOutcome["kind"][] = [];
  const unsubscribe = subscribeApiOutcome((outcome) =>
    outcomeList.push(outcome.kind),
  );
  return { outcomeList, unsubscribe };
};

beforeEach(() => {
  vi.restoreAllMocks();
  resetAuthStore();
  useApiStatusStore.setState({ unreachable: false, reason: null });
});

afterEach(() => {
  vi.useRealTimers();
});

describe("apiJson classification", () => {
  it("wraps a connection failure in ApiUnreachableError", async () => {
    // What the browser actually throws when the server is not listening.
    stubFetchSequence([failedFetch]);

    const error = await apiJson("/example").catch((err: unknown) => err);

    expect(error).toBeInstanceOf(ApiUnreachableError);
    expect((error as ApiUnreachableError).status).toBeUndefined();
  });

  it("preserves the original failure as the cause", async () => {
    const original = new TypeError("Failed to fetch");
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(original));

    await expect(apiJson("/example")).rejects.toMatchObject({
      cause: original,
    });
  });

  it.each([500, 502, 503, 429])(
    "throws ApiUnreachableError for %i and announces it",
    async (status) => {
      stubFetchSequence([answerWith(status)]);
      const { outcomeList, unsubscribe } = recordOutcomes();

      const error = await apiJson("/example").catch((err: unknown) => err);
      unsubscribe();

      expect(error).toBeInstanceOf(ApiUnreachableError);
      expect((error as ApiUnreachableError).status).toBe(status);
      expect(outcomeList).toEqual(["unreachable"]);
    },
  );

  it("carries the server's message on a 429 (an account lockout)", async () => {
    stubFetchSequence([
      () =>
        Promise.resolve(
          json(429, { error: "Account is temporarily locked" }),
        ),
    ]);

    await expect(
      apiJson("/auth", { method: "POST" }, { anonymous: true }),
    ).rejects.toThrow("Account is temporarily locked");
  });

  it("does NOT classify a 401 as unreachable", async () => {
    // The load-bearing half. A 401 is a real answer from a reachable server
    // and travels as SessionExpiredError, not as an outage.
    stubFetchSequence([answerWith(401)]);
    const { outcomeList, unsubscribe } = recordOutcomes();

    const error = await apiJson("/example").catch((err: unknown) => err);
    unsubscribe();

    expect(error).toBeInstanceOf(SessionExpiredError);
    expect(isApiUnreachable(error)).toBe(false);
    expect(outcomeList).toEqual(["session-expired"]);
  });

  it("announces ok on a 2xx", async () => {
    stubFetchSequence([() => Promise.resolve(json(200, { data: [] }))]);
    const { outcomeList, unsubscribe } = recordOutcomes();

    await apiJson("/example");
    unsubscribe();

    expect(outcomeList).toEqual(["ok"]);
  });

  it("rethrows an abort unchanged rather than calling it an outage", async () => {
    // Navigating away from a slow page cancels its requests. Reporting that
    // as a server failure would put an outage banner in front of somebody
    // whose server is perfectly healthy.
    const abort = new DOMException("The operation was aborted.", "AbortError");
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(abort));
    const { outcomeList, unsubscribe } = recordOutcomes();

    await expect(apiJson("/example")).rejects.toBe(abort);
    unsubscribe();

    expect(isApiUnreachable(abort)).toBe(false);
    expect(outcomeList).toEqual([]);
  });
});

describe("api status store", () => {
  it("is raised once by many failures and cleared by the next ok", async () => {
    const renderList: boolean[] = [];
    const unsubscribe = useApiStatusStore.subscribe((state) =>
      renderList.push(state.unreachable),
    );
    stubFetchSequence([
      answerWith(502),
      answerWith(503),
      failedFetch,
      () => Promise.resolve(json(200, {})),
      () => Promise.resolve(json(200, {})),
    ]);

    for (let call = 0; call < 3; call++) {
      await apiJson("/example").catch(() => undefined);
    }
    expect(useApiStatusStore.getState().unreachable).toBe(true);
    expect(useApiStatusStore.getState().reason).toMatch(/502/);

    await apiJson("/example");
    await apiJson("/example");
    unsubscribe();

    expect(useApiStatusStore.getState().unreachable).toBe(false);
    // One change up, one change down: three failures render the banner once.
    expect(renderList).toEqual([true, false]);
  });

  it("is not raised by a refusal", async () => {
    stubFetchSequence([answerWith(403), answerWith(404)]);

    await apiJson("/example").catch(() => undefined);
    await apiJson("/example").catch(() => undefined);

    expect(useApiStatusStore.getState().unreachable).toBe(false);
  });
});

describe("auth store: unreachable vs unauthenticated", () => {
  // The page-load /refresh retries an outage at 1, 2 and 4 seconds.
  const runRefresh = async () => {
    vi.useFakeTimers();
    const refresh = useAuthStore.getState().authenticateWithAuthToken();
    await vi.runAllTimersAsync();
    await refresh;
  };

  it("retries an unreachable /refresh and ends YES when it answers", async () => {
    const fetchMock = stubFetchSequence([
      answerWith(502),
      answerWith(502),
      () => Promise.resolve(refreshAnswer()),
    ]);

    await runRefresh();

    expect(fetchMock).toHaveBeenCalledTimes(3);
    expect(useAuthStore.getState().authState).toBe("YES");
    expect(useApiStatusStore.getState().unreachable).toBe(false);
  });

  it("ends UNREACHABLE after four failures and seven seconds", async () => {
    vi.useFakeTimers();
    const fetchMock = stubFetchSequence([
      answerWith(502),
      answerWith(502),
      answerWith(502),
      answerWith(502),
    ]);

    const refresh = useAuthStore.getState().authenticateWithAuthToken();
    await vi.advanceTimersByTimeAsync(6999);
    expect(useAuthStore.getState().authState).toBe("CHECKING");
    await vi.advanceTimersByTimeAsync(1);
    await refresh;

    expect(fetchMock).toHaveBeenCalledTimes(4);
    expect(useAuthStore.getState().authState).toBe("UNREACHABLE");
  });

  it("reports UNREACHABLE when the session check cannot reach the server", async () => {
    stubFetchSequence([failedFetch, failedFetch, failedFetch, failedFetch]);

    await runRefresh();

    // NOT "NO". Answering "NO" here is the bug: it means the app claims to
    // know the person is signed out when it has learned nothing at all, and
    // sends them to a login form that cannot work either.
    expect(useAuthStore.getState().authState).toBe("UNREACHABLE");
  });

  it.each([500, 503, 504, 429])(
    "reports UNREACHABLE when every attempt answers %i",
    async (status) => {
      // nginx answers 502 while the Go server restarts, and Vite's dev proxy
      // does the same; nobody asked the server about the session.
      stubFetchSequence([1, 2, 3, 4].map(() => answerWith(status)));

      await runRefresh();

      expect(useAuthStore.getState().authState).toBe("UNREACHABLE");
    },
  );

  it("reports NO on a 401, with no retry and no toast", async () => {
    const toast = vi.spyOn(notifications, "show");
    const fetchMock = stubFetchSequence([answerWith(401)]);

    await runRefresh();

    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(useAuthStore.getState().authState).toBe("NO");
    expect(toast).not.toHaveBeenCalled();
  });

  it("reports NO on a 403, which IS the server refusing this person", async () => {
    stubFetchSequence([answerWith(403)]);

    await runRefresh();

    expect(useAuthStore.getState().authState).toBe("NO");
  });

  it("does not claim a session ended because the body would not parse", async () => {
    // An HTML error page from a proxy arriving with a 200 is the shape here.
    stubFetchSequence([
      () =>
        Promise.resolve(
          new Response("<html>Bad gateway</html>", { status: 200 }),
        ),
    ]);

    await runRefresh();

    expect(useAuthStore.getState().authState).toBe("UNREACHABLE");
  });

  it("does not claim a session ended on another refusal", async () => {
    // Only 401/403 say "not signed in"; a 404 (a proxy routing /api wrong)
    // says nothing about the session and is not retried.
    const fetchMock = stubFetchSequence([answerWith(404)]);

    await runRefresh();

    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(useAuthStore.getState().authState).toBe("UNREACHABLE");
  });

  it("reports YES on a successful session check", async () => {
    stubFetchSequence([() => Promise.resolve(refreshAnswer())]);

    await runRefresh();

    expect(useAuthStore.getState().authState).toBe("YES");
  });

  it("shows no session toast for a 401 while CHECKING", async () => {
    // Every page load asks before it knows whether anyone is signed in; a
    // 401 announced then is someone who was never signed in.
    useAuthStore.setState({ authState: "CHECKING" });
    const toast = vi.spyOn(notifications, "show");
    stubFetchSequence([answerWith(401)]);

    await expect(apiJson("/example")).rejects.toBeInstanceOf(
      SessionExpiredError,
    );

    expect(toast).not.toHaveBeenCalled();
    expect(useAuthStore.getState().authState).toBe("CHECKING");
  });

  it("keeps a failed SIGN-IN on the login form, with an accurate message", async () => {
    // Deliberately still "NO" rather than "UNREACHABLE": the person is
    // standing at the form, and replacing it with an outage screen would
    // take away the thing they need to retry.
    stubFetchSequence([failedFetch]);

    await useAuthStore
      .getState()
      .authenticateWithUsernamePassword("someone@example.com", "pw");

    const state = useAuthStore.getState();
    expect(state.authState).toBe("NO");
    expect(state.error).toMatch(/could not reach the server/i);
  });

  it("tells a locked-out SIGN-IN the server's reason", async () => {
    stubFetchSequence([
      () =>
        Promise.resolve(
          json(429, {
            error:
              "Account is temporarily locked due to too many failed login " +
              "attempts",
          }),
        ),
    ]);

    await useAuthStore
      .getState()
      .authenticateWithUsernamePassword("someone@example.com", "pw");

    const state = useAuthStore.getState();
    expect(state.authState).toBe("NO");
    expect(state.error).toMatch(/temporarily locked/);
  });

  it("does not blame the connection for a rejected password", async () => {
    stubFetchSequence([answerWith(401)]);

    await useAuthStore
      .getState()
      .authenticateWithUsernamePassword("someone@example.com", "wrong");

    const state = useAuthStore.getState();
    expect(state.authState).toBe("NO");
    expect(state.error).not.toMatch(/could not reach the server/i);
  });
});
