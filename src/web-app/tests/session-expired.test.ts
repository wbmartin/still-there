import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { notifications } from "@mantine/notifications";
import {
  ApiError,
  SessionExpiredError,
  apiJson,
  isSessionExpired,
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

// Every call answers with this status (a fresh Response each time).
const answerAll = (status: number, body: unknown = {}) => {
  vi.stubGlobal(
    "fetch",
    vi.fn(() => Promise.resolve(responseWith(status, body))),
  );
};

const callWith = (status: number, body: unknown = {}) => {
  answerAll(status, body);
  return apiJson<unknown>("/example");
};

describe("apiJson session handling", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    resetAuthStore();
  });
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("throws SessionExpiredError and signs the app out on 401", async () => {
    const toast = vi.spyOn(notifications, "show");
    await expect(callWith(401)).rejects.toBeInstanceOf(SessionExpiredError);

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
    const error = await callWith(403, { error: "Permission denied" }).catch(
      (error: unknown) => error,
    );

    expect(error).toBeInstanceOf(ApiError);
    expect((error as ApiError).status).toBe(403);
    expect(error).not.toBeInstanceOf(SessionExpiredError);
    expect((error as Error).message).toBe("Permission denied");
    expect(toast).not.toHaveBeenCalled();
    expect(useAuthStore.getState().authState).toBe("YES");
    expect(useAuthStore.getState().userName).toBe("someone@example.com");
  });

  it("leaves other failures alone", async () => {
    // A 500 says nothing about the session. Signing the user out here is the
    // original bug in mirror image -- an outage reported as an expiry.
    await expect(callWith(500)).rejects.toThrow(Error);
    await expect(callWith(500)).rejects.not.toBeInstanceOf(
      SessionExpiredError,
    );

    expect(useAuthStore.getState().authState).toBe("YES");
  });

  it("carries the server's message", async () => {
    await expect(
      callWith(409, { error: "Someone else changed this first" }),
    ).rejects.toThrow("Someone else changed this first");
  });

  it("does not stomp state when several requests fail at once", async () => {
    // A page with four in-flight reads gets four 401s. The first does the work;
    // the rest must not re-clear already-cleared state.
    await expect(callWith(401)).rejects.toThrow();
    useAuthStore.setState({ error: "a later, more specific message" });

    await expect(callWith(401)).rejects.toThrow();

    expect(useAuthStore.getState().error).toBe("a later, more specific message");
  });

  it("leaves the session alone on an anonymous 401", async () => {
    // /auth's wrong password: an ApiError for the form, not a sign-out.
    const toast = vi.spyOn(notifications, "show");
    answerAll(401, { error: "Invalid credentials" });

    const error = await apiJson<unknown>(
      "/auth",
      { method: "POST" },
      { anonymous: true },
    ).catch((error: unknown) => error);

    expect(error).toBeInstanceOf(ApiError);
    expect(error).not.toBeInstanceOf(SessionExpiredError);
    expect((error as ApiError).message).toBe("Invalid credentials");
    expect(toast).not.toHaveBeenCalled();
    expect(useAuthStore.getState().authState).toBe("YES");
  });

  it("identifies its own error type", () => {
    expect(isSessionExpired(new SessionExpiredError(401))).toBe(true);
    expect(isSessionExpired(new Error("something else"))).toBe(false);
  });
});
