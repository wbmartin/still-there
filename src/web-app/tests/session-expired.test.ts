import { beforeEach, describe, expect, it, vi } from "vitest";
import { notifications } from "@mantine/notifications";
import {
  SessionExpiredError,
  isSessionExpired,
  throwForFailedResponse,
} from "../utils/api";
import { useAuthStore } from "../stores/auth-store";

// The gap these tests protect: every mutation used to throw a bare Error on
// !response.ok, so a 401 was indistinguishable from any other failure.
// authState stayed "YES", the UI carried on rendering as signed in, and the
// user saw an unexplained error rather than a prompt to sign in again.
//
// Worth a test rather than a careful reading for the same reason as its
// sibling in api-unreachable.test.ts: the reporting runs through a registered
// listener to avoid an import cycle between utils/api and the auth store, and a
// listener that silently stops being registered breaks nothing a typecheck can
// see. It fails only in front of a user, on the one path where they most need
// to be told what happened.

const resetAuthStore = () => {
  useAuthStore.setState({
    authState: "YES",
    grantList: [],
    availableSecurityContextList: [],
    userName: "someone@example.com",
    securityContext: 1,
    loading: false,
    error: null,
  });
};

const responseWith = (status: number, body: unknown = {}) =>
  new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });

const readMessage = async (response: Response) => {
  const body = (await response.json()) as { error?: string };
  return body.error || `Request failed (${response.status})`;
};

describe("throwForFailedResponse", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    resetAuthStore();
  });

  it("throws SessionExpiredError and signs the app out on 401", async () => {
    const toast = vi.spyOn(notifications, "show");
    await expect(
      throwForFailedResponse(responseWith(401), readMessage),
    ).rejects.toBeInstanceOf(SessionExpiredError);

    expect(useAuthStore.getState().authState).toBe("NO");
    expect(useAuthStore.getState().userName).toBe("");
    expect(toast).toHaveBeenCalledOnce();
  });

  it("keeps the session on a 403 and carries its reason", async () => {
    // A 403 is a refusal (grant, blocked, CSRF, tenant); signing in again
    // fixes none of them, so it must not sign the user out. The store's
    // session-expired listener is the only caller of the "Session ended"
    // toast, so no toast means it never ran.
    const toast = vi.spyOn(notifications, "show");
    const error = await throwForFailedResponse(
      responseWith(403, { error: "Permission denied" }),
      readMessage,
    ).catch((error: unknown) => error);

    expect(error).toBeInstanceOf(Error);
    expect(error).not.toBeInstanceOf(SessionExpiredError);
    expect((error as Error).message).toBe("Permission denied");
    expect(toast).not.toHaveBeenCalled();
    expect(useAuthStore.getState().authState).toBe("YES");
    expect(useAuthStore.getState().userName).toBe("someone@example.com");
  });

  it("leaves other failures alone", async () => {
    // A 500 says nothing about the session. Signing the user out here is the
    // original bug in mirror image -- an outage reported as an expiry.
    await expect(
      throwForFailedResponse(responseWith(500), readMessage),
    ).rejects.toThrow(Error);
    await expect(
      throwForFailedResponse(responseWith(500), readMessage),
    ).rejects.not.toBeInstanceOf(SessionExpiredError);

    expect(useAuthStore.getState().authState).toBe("YES");
  });

  it("carries the server's message", async () => {
    await expect(
      throwForFailedResponse(
        responseWith(409, { error: "Someone else changed this first" }),
        readMessage,
      ),
    ).rejects.toThrow("Someone else changed this first");
  });

  it("does not stomp state when several requests fail at once", async () => {
    // A page with four in-flight reads gets four 401s. The first does the work;
    // the rest must not re-clear already-cleared state.
    await expect(
      throwForFailedResponse(responseWith(401), readMessage),
    ).rejects.toThrow();
    useAuthStore.setState({ error: "a later, more specific message" });

    await expect(
      throwForFailedResponse(responseWith(401), readMessage),
    ).rejects.toThrow();

    expect(useAuthStore.getState().error).toBe("a later, more specific message");
  });

  it("identifies its own error type", () => {
    expect(isSessionExpired(new SessionExpiredError(401))).toBe(true);
    expect(isSessionExpired(new Error("something else"))).toBe(false);
  });
});
