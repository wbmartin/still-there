import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { notifications } from "@mantine/notifications";
import {
  ApiError,
  apiJson,
  clearCsrfToken,
  isApiStatus,
  setCsrfToken,
} from "../utils/api";
import { createEntityStore } from "../stores/create-entity-store";
import { useExamplesStore } from "../stores/examples-store";
import { useAuthStore } from "../stores/auth-store";

// apiJson is the one way to the API (B037): the raw request function is
// private, so every rule about a status lives in one place. These pin what
// the stores now rely on it for -- the CSRF header only on a session call,
// a refusal's status on the error, and an entity write's request shape.

vi.mock("@mantine/notifications", () => ({
  notifications: { show: vi.fn() },
}));

const json = (status: number, body?: unknown) =>
  new Response(body === undefined ? null : JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });

type FetchMock = ReturnType<typeof vi.fn<
  (input: string, init?: RequestInit) => Promise<Response>
>>;

const stubFetch = (...answerList: Response[]): FetchMock => {
  const fetchMock = vi.fn<
    (input: string, init?: RequestInit) => Promise<Response>
  >(() => {
    const answer = answerList.shift();
    if (!answer) throw new Error("unexpected fetch");
    return Promise.resolve(answer);
  });
  vi.stubGlobal("fetch", fetchMock);
  return fetchMock;
};

const sentHeaders = (fetchMock: FetchMock, call = 0) =>
  new Headers(fetchMock.mock.calls[call][1]?.headers);

interface Widget {
  widgetId: number;
  label: string;
  updateTs: number;
}

const buildWidgetStore = () =>
  createEntityStore<Widget>({
    idParamName: "widgetId",
    idField: "widgetId",
    listEndpoint: "/widgetList",
    itemEndpoint: "/widget",
  });

describe("apiJson", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    setCsrfToken("session-token");
    useAuthStore.setState({ authState: "YES", userName: "alice" });
  });
  afterEach(() => {
    vi.unstubAllGlobals();
    clearCsrfToken();
  });

  it("sends the CSRF header on a session call", async () => {
    const fetchMock = stubFetch(json(200, { data: {} }));

    await apiJson("/example", { method: "POST" });

    expect(sentHeaders(fetchMock).get("X-CSRF-Token")).toBe("session-token");
  });

  it("withholds the CSRF header from an anonymous call", async () => {
    // A token left from an ended session must not ride on /auth.
    const fetchMock = stubFetch(json(200, { user: {} }));

    await apiJson("/auth", { method: "POST" }, { anonymous: true });

    expect(sentHeaders(fetchMock).has("X-CSRF-Token")).toBe(false);
  });

  it.each([
    ["an empty 200", 200],
    ["a 204", 204],
  ])("returns undefined for %s", async (_label, status) => {
    stubFetch(json(status));

    await expect(apiJson("/logoff", { method: "POST" })).resolves.toBe(
      undefined,
    );
  });

  it("puts the status and the server's message on a refusal", async () => {
    stubFetch(json(409, { error: "Someone else changed this first" }));

    const error = await apiJson("/example").catch((error: unknown) => error);

    expect(error).toBeInstanceOf(ApiError);
    expect((error as ApiError).status).toBe(409);
    expect((error as ApiError).message).toBe(
      "Someone else changed this first",
    );
    expect(isApiStatus(error, 404, 409)).toBe(true);
    expect(isApiStatus(error, 404)).toBe(false);
  });
});

describe("entity store writes through apiJson", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    setCsrfToken("session-token");
  });
  afterEach(() => {
    vi.unstubAllGlobals();
    clearCsrfToken();
  });

  it("creates, updates and deletes with the right request", async () => {
    const useStore = buildWidgetStore();
    const fetchMock = stubFetch(
      json(200, { data: { widgetId: 7, label: "new", updateTs: 10 } }),
      json(200, { data: { widgetId: 7, label: "edited", updateTs: 20 } }),
      json(200),
    );

    await useStore.getState().createItem({ label: "new" });
    await useStore.getState().updateItem({ widgetId: 7, label: "edited" });
    expect(useStore.getState().itemList).toEqual([
      { widgetId: 7, label: "edited", updateTs: 20 },
    ]);
    await useStore.getState().deleteItem(7, 20);

    const [create, update, remove] = fetchMock.mock.calls;
    expect(create[0]).toBe("/api/widget");
    expect(create[1]?.method).toBe("POST");
    expect(create[1]?.body).toBe(JSON.stringify({ label: "new" }));
    expect(update[1]?.method).toBe("PUT");
    expect(update[1]?.body).toBe(
      JSON.stringify({ widgetId: 7, label: "edited" }),
    );
    expect(remove[0]).toBe("/api/widget?widgetId=7&updateTs=20");
    expect(remove[1]?.method).toBe("DELETE");
    expect(remove[1]?.body).toBeUndefined();
    [0, 1].forEach((call) => {
      expect(sentHeaders(fetchMock, call).get("Content-Type")).toBe(
        "application/json",
      );
    });
    [0, 1, 2].forEach((call) => {
      expect(sentHeaders(fetchMock, call).get("X-CSRF-Token")).toBe(
        "session-token",
      );
    });
    expect(useStore.getState().itemList).toEqual([]);
  });

  it("leaves the cache alone when a write is refused", async () => {
    const useStore = buildWidgetStore();
    stubFetch(json(409, { error: "Stale update" }));

    await expect(
      useStore.getState().updateItem({ widgetId: 7, label: "x" }),
    ).rejects.toThrow("Stale update");

    expect(useStore.getState().itemList).toEqual([]);
    expect(notifications.show).not.toHaveBeenCalled();
  });
});

describe("examples store readExampleById", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it.each([404, 400])("reads a %i as no such row", async (status) => {
    stubFetch(json(status, { error: "Example not found" }));

    await expect(
      useExamplesStore.getState().readExampleById(99),
    ).resolves.toBeNull();
  });

  it("throws any other refusal", async () => {
    stubFetch(json(403, { error: "Permission denied" }));

    await expect(
      useExamplesStore.getState().readExampleById(99),
    ).rejects.toThrow("Permission denied");
  });
});

describe("logout", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
    clearCsrfToken();
  });

  it("signs out quietly when /logoff finds the session gone", async () => {
    // /logoff is anonymous: its 401 must not raise "Session ended" over a
    // sign-out the person asked for.
    vi.clearAllMocks();
    setCsrfToken("session-token");
    useAuthStore.setState({ authState: "YES", userName: "alice" });
    const fetchMock = stubFetch(json(401, { error: "Unauthorized" }));

    await useAuthStore.getState().logout();

    expect(sentHeaders(fetchMock).has("X-CSRF-Token")).toBe(false);
    expect(notifications.show).not.toHaveBeenCalled();
    expect(useAuthStore.getState().authState).toBe("NO");
  });
});
