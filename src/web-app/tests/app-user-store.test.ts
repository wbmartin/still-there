import { afterEach, describe, expect, it, vi } from "vitest";
import { useAppUserStore } from "../stores/app-user-store";

// A body key the handler does not read is dropped by Go's JSON decoder, so
// a misnamed key reaches the server as a zero and fails every call (B013).
// The keys here are the json tags in handler/user_handler.go.

const recordFetch = () => {
  const fetchMock = vi.fn<
    (input: string, init?: RequestInit) => Promise<Response>
  >(() =>
    Promise.resolve(
      new Response(JSON.stringify({ data: { message: "ok" } }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    ),
  );
  vi.stubGlobal("fetch", fetchMock);
  return fetchMock;
};

describe("app user store request bodies", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("changeOtherUserPassword posts appUserId and newPassword", async () => {
    const fetchMock = recordFetch();

    await useAppUserStore
      .getState()
      .changeOtherUserPassword(51, "n3w-passw0rd!");

    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toBe("/api/changeOtherUserPassword");
    expect(init?.method).toBe("POST");
    expect(JSON.parse(init?.body as string)).toEqual({
      appUserId: 51,
      newPassword: "n3w-passw0rd!",
    });
  });
});
