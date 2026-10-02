import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useAuthStore } from "../stores/auth-store";

// /auth says whether the temp token's user is enrolled, and the login page
// routes on it: an enrolled user verifies a code and never asks /mfaSetup,
// which used to hand back the stored secret to anyone with the password.

const tempTokenAnswer = (mfaEnrolled?: boolean) =>
  new Response(
    JSON.stringify({
      user: {
        userName: "mfa",
        tempToken: "temp-123",
        grantList: [],
        availableSecurityContextList: [],
        securityContext: 1,
        ...(mfaEnrolled === undefined ? {} : { mfaEnrolled }),
      },
    }),
    { status: 200, headers: { "Content-Type": "application/json" } },
  );

describe("authenticateWithUsernamePassword, a second factor owed", () => {
  beforeEach(() => {
    useAuthStore.setState({
      authState: "NO",
      mfaRequired: false,
      tempToken: undefined,
      mfaEnrolled: false,
      loading: false,
      error: null,
    });
  });
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("an enrolled user is MFA with mfaEnrolled true", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(tempTokenAnswer(true)));
    await useAuthStore.getState().authenticateWithUsernamePassword("mfa", "pw");
    const state = useAuthStore.getState();
    expect(state.authState).toBe("MFA");
    expect(state.tempToken).toBe("temp-123");
    expect(state.mfaEnrolled).toBe(true);
  });

  it("a user still to enrol is MFA with mfaEnrolled false", async () => {
    useAuthStore.setState({ mfaEnrolled: true });
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(tempTokenAnswer(false)));
    await useAuthStore.getState().authenticateWithUsernamePassword("mfa", "pw");
    expect(useAuthStore.getState().authState).toBe("MFA");
    expect(useAuthStore.getState().mfaEnrolled).toBe(false);
  });

  it("an answer without mfaEnrolled is treated as enrolment", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(tempTokenAnswer()));
    await useAuthStore.getState().authenticateWithUsernamePassword("mfa", "pw");
    expect(useAuthStore.getState().mfaEnrolled).toBe(false);
  });
});
