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

  it.each([
    [true, "/api/admin/blockUser"],
    [false, "/api/admin/unblockUser"],
  ])("setUserBlocked(%s) posts userName to %s", async (blocked, path) => {
    const fetchMock = recordFetch();

    await useAppUserStore.getState().setUserBlocked("bob", blocked);

    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toBe(path);
    expect(init?.method).toBe("POST");
    expect(JSON.parse(init?.body as string)).toEqual({ userName: "bob" });
  });

  it("inviteUser posts the user fields, no password, and returns the token",
    async () => {
      const fetchMock = vi.fn<
        (input: string, init?: RequestInit) => Promise<Response>
      >(() =>
        Promise.resolve(
          new Response(
            JSON.stringify({
              data: { inviteToken: "ab".repeat(32), expiresTs: 1800000000 },
            }),
            { status: 201, headers: { "Content-Type": "application/json" } },
          ),
        ),
      );
      vi.stubGlobal("fetch", fetchMock);
      const data = {
        userName: "ivy",
        firstName: "Ivy",
        lastName: "Vine",
        email: "ivy@example.com",
        appUserGroupMembershipList: [1],
        mfaExempt: false,
      };

      const issued = await useAppUserStore.getState().inviteUser(data);

      const [url, init] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/userInvite");
      expect(init?.method).toBe("POST");
      expect(JSON.parse(init?.body as string)).toEqual(data);
      expect(issued).toEqual({
        userName: "ivy",
        inviteToken: "ab".repeat(32),
        expiresTs: 1800000000,
      });
    });

  it.each([
    ["reissueInvite", "/api/userInviteReissue"],
    ["revokeInvite", "/api/userInviteRevoke"],
  ] as const)("%s posts appUserId to %s", async (action, path) => {
    const fetchMock = recordFetch();

    if (action === "reissueInvite") {
      await useAppUserStore.getState().reissueInvite(51, "ivy");
    } else {
      await useAppUserStore.getState().revokeInvite(51);
    }

    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toBe(path);
    expect(init?.method).toBe("POST");
    expect(JSON.parse(init?.body as string)).toEqual({ appUserId: 51 });
  });
});
