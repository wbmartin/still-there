import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { notifications } from "@mantine/notifications";
import {
  ApiError,
  SessionChangedError,
  apiJson,
  getCsrfToken,
  setCsrfToken,
} from "../utils/api";
import { setRouteNavigator, useAuthStore } from "../stores/auth-store";
import { createEntityStore } from "../stores/create-entity-store";
import { registerCacheClearer } from "../utils/cache-utils";
import { ROUTES } from "../constants/routes";

// Every tab shares the session cookie but holds its own CSRF token. When
// another tab signs in or switches tenant the cookie moves to a new session
// and this tab's next write is refused 403 "code":"CSRF". apiJson heals
// that once through /refresh -- unless the cookie now belongs to someone
// else, in which case the write is dropped and the tab moves to them.

vi.mock("@mantine/notifications", () => ({
  notifications: { show: vi.fn() },
}));

const json = (status: number, body: unknown) =>
  new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });

const csrfRefusal = () =>
  json(403, { error: "CSRF validation failed", code: "CSRF" });

const refreshAnswer = (userName = "alice", securityContext = 1) =>
  json(200, {
    user: {
      userName,
      securityContext,
      grantList: [7],
      availableSecurityContextList: [1, 2],
    },
    csrfToken: "new",
  });

type Route = (method: string) => Response | Promise<Response>;

// fetch answering by path; each path's answers are used in order.
const stubFetch = (routes: Record<string, Route[]>) => {
  const fetchMock = vi.fn((input: string, init?: RequestInit) => {
    const path = input.replace(/^\/api/, "").split("?")[0];
    const answer = routes[path]?.shift();
    if (!answer) throw new Error(`unexpected fetch ${path}`);
    return Promise.resolve(answer(init?.method || "GET"));
  });
  vi.stubGlobal("fetch", fetchMock);
  return fetchMock;
};

const calledPaths = (fetchMock: ReturnType<typeof stubFetch>) =>
  fetchMock.mock.calls.map(([input]) => input);

const sentToken = (fetchMock: ReturnType<typeof stubFetch>, call: number) =>
  new Headers(fetchMock.mock.calls[call][1]?.headers).get("X-CSRF-Token");

// The status a call was refused with, or 200 when it resolved.
const statusOf = (call: Promise<unknown>) =>
  call.then(
    () => 200,
    (error: unknown) => {
      if (error instanceof ApiError) return error.status;
      throw error;
    },
  );

const send = (url: string, options: RequestInit = {}) =>
  statusOf(apiJson<unknown>(url, options));

const navigateSpy = vi.fn();
const cacheClearSpy = vi.fn();
registerCacheClearer(cacheClearSpy);

describe("apiJson CSRF resync", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    setRouteNavigator(navigateSpy);
    setCsrfToken("old");
    useAuthStore.setState({
      authState: "YES",
      userName: "alice",
      securityContext: 1,
      grantList: [7],
      availableSecurityContextList: [1, 2],
      error: null,
    });
  });
  afterEach(() => {
    vi.unstubAllGlobals();
    setRouteNavigator(null);
  });

  it("refreshes the token and resends the write once", async () => {
    const fetchMock = stubFetch({
      "/example": [() => csrfRefusal(), () => json(200, { data: {} })],
      "/refresh": [() => refreshAnswer()],
    });

    const status = await send("/example", { method: "POST" });

    expect(status).toBe(200);
    expect(calledPaths(fetchMock)).toEqual([
      "/api/example",
      "/api/refresh",
      "/api/example",
    ]);
    expect(sentToken(fetchMock, 0)).toBe("old");
    expect(sentToken(fetchMock, 2)).toBe("new");
    expect(getCsrfToken()).toBe("new");
    expect(navigateSpy).not.toHaveBeenCalled();
  });

  it("retries at most once; a second refusal comes back as is", async () => {
    const fetchMock = stubFetch({
      "/example": [() => csrfRefusal(), () => csrfRefusal()],
      "/refresh": [() => refreshAnswer()],
    });

    const status = await send("/example", { method: "PUT" });

    expect(status).toBe(403);
    expect(fetchMock).toHaveBeenCalledTimes(3);
  });

  it("leaves a refused GET alone", async () => {
    const fetchMock = stubFetch({ "/example": [() => csrfRefusal()] });

    const status = await send("/example");

    expect(status).toBe(403);
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it("leaves a 403 without the CSRF code alone", async () => {
    const fetchMock = stubFetch({
      "/example": [() => json(403, { error: "Access denied" })],
    });

    const status = await send("/example", { method: "DELETE" });

    expect(status).toBe(403);
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it("does not resend when the refresh fails", async () => {
    const fetchMock = stubFetch({
      "/example": [() => csrfRefusal()],
      "/refresh": [() => json(401, { error: "Unauthorized" })],
    });

    const status = await send("/example", { method: "POST" });

    expect(status).toBe(403);
    expect(fetchMock).toHaveBeenCalledTimes(2);
    expect(getCsrfToken()).toBe("old");
  });

  it("two refused writes share one /refresh", async () => {
    const fetchMock = stubFetch({
      "/example": [
        () => csrfRefusal(),
        () => csrfRefusal(),
        () => json(200, {}),
        () => json(200, {}),
      ],
      "/refresh": [() => refreshAnswer()],
    });

    const [first, second] = await Promise.all([
      send("/example", { method: "POST" }),
      send("/example", { method: "POST" }),
    ]);

    expect(first).toBe(200);
    expect(second).toBe(200);
    expect(
      calledPaths(fetchMock).filter((path) => path === "/api/refresh"),
    ).toHaveLength(1);
  });

  it.each([
    ["another user", "bob", 1],
    ["another tenant", "alice", 2],
  ])(
    "drops the write and moves the tab to %s",
    async (_label, userName, securityContext) => {
      const fetchMock = stubFetch({
        "/example": [() => csrfRefusal(), () => csrfRefusal()],
        "/refresh": [() => refreshAnswer(userName, securityContext)],
      });

      await Promise.all([
        expect(
          apiJson("/example", { method: "POST" }),
        ).rejects.toBeInstanceOf(SessionChangedError),
        expect(
          apiJson("/example", { method: "POST" }),
        ).rejects.toBeInstanceOf(SessionChangedError),
      ]);

      expect(calledPaths(fetchMock)).toEqual([
        "/api/example",
        "/api/example",
        "/api/refresh",
      ]);
      const state = useAuthStore.getState();
      expect(state.authState).toBe("YES");
      expect(state.userName).toBe(userName);
      expect(state.securityContext).toBe(securityContext);
      expect(getCsrfToken()).toBe("new");
      expect(cacheClearSpy).toHaveBeenCalledTimes(1);
      expect(navigateSpy).toHaveBeenCalledTimes(1);
      expect(navigateSpy).toHaveBeenCalledWith(ROUTES.DASHBOARD);
      expect(notifications.show).toHaveBeenCalledTimes(1);
    },
  );

  it("heals an entity-store write", async () => {
    const useStore = createEntityStore<{ widgetId: number; updateTs: number }>(
      {
        idParamName: "widgetId",
        idField: "widgetId",
        listEndpoint: "/widgetList",
        itemEndpoint: "/widget",
      },
    );
    const fetchMock = stubFetch({
      "/widget": [
        () => csrfRefusal(),
        () => json(200, { data: { widgetId: 9, updateTs: 1 } }),
      ],
      "/refresh": [() => refreshAnswer()],
    });

    const created = await useStore.getState().createItem({ updateTs: 0 });

    expect(created.widgetId).toBe(9);
    expect(fetchMock).toHaveBeenCalledTimes(3);
    expect(useAuthStore.getState().authState).toBe("YES");
  });
});

describe("changeSecurityContext", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("keeps the new session's CSRF token", async () => {
    setCsrfToken("old");
    stubFetch({
      "/changeSecurityContext": [
        () =>
          json(200, {
            user: {
              userName: "alice",
              securityContext: 2,
              grantList: [],
              availableSecurityContextList: [1, 2],
            },
            csrfToken: "ctx",
          }),
      ],
    });

    await useAuthStore.getState().changeSecurityContext(2);

    expect(getCsrfToken()).toBe("ctx");
    expect(useAuthStore.getState().securityContext).toBe(2);
  });
});
