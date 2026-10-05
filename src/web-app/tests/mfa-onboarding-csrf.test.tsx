import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { MemoryRouter } from "react-router-dom";
import { MantineProvider } from "@mantine/core";
import MFAOnboardingPage from "../pages/mfa-onboarding-page";
import { useAuthStore } from "../stores/auth-store";
import { apiJson, clearCsrfToken, getCsrfToken } from "../utils/api";

// /mfaVerify mints the session and answers with its CSRF token, as /auth
// does. The page must store it: without it the first write after MFA is
// refused 403 and only the /refresh retry in api.ts hides it.

vi.mock("@mantine/notifications", () => ({
  notifications: { show: vi.fn() },
}));

const json = (status: number, body: unknown) =>
  new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });

const mfaVerifyAnswer = () =>
  json(200, {
    user: {
      userName: "mfa",
      securityContext: 1,
      grantList: [7],
      availableSecurityContextList: [1],
    },
    csrfToken: "mfa-session-token",
  });

describe("MFAOnboardingPage, a verified code", () => {
  let container: HTMLDivElement;
  let root: Root;

  beforeEach(() => {
    vi.stubGlobal("IS_REACT_ACT_ENVIRONMENT", true);
    // jsdom has no matchMedia; MantineProvider reads the colour scheme.
    vi.stubGlobal("matchMedia", () => ({
      matches: false,
      addEventListener() {},
      removeEventListener() {},
    }));
    clearCsrfToken();
    // Enrolled: the page only verifies, so /mfaSetup is never asked.
    useAuthStore.setState({
      authState: "MFA",
      mfaRequired: true,
      tempToken: "temp-123",
      mfaEnrolled: true,
    });
    container = document.createElement("div");
    document.body.appendChild(container);
    root = createRoot(container);
  });

  afterEach(() => {
    act(() => root.unmount());
    container.remove();
    vi.unstubAllGlobals();
  });

  it("stores the CSRF token, and the next write carries it", async () => {
    const fetchMock = vi
      .fn<(input: string, init?: RequestInit) => Promise<Response>>()
      .mockResolvedValueOnce(mfaVerifyAnswer())
      .mockResolvedValueOnce(json(200, {}));
    vi.stubGlobal("fetch", fetchMock);

    act(() =>
      root.render(
        <MantineProvider>
          <MemoryRouter>
            <MFAOnboardingPage />
          </MemoryRouter>
        </MantineProvider>,
      ),
    );
    const verifyButton = Array.from(container.querySelectorAll("button"))
      .find((button) => button.textContent === "Verify Code");
    expect(verifyButton).toBeDefined();
    // handleVerify is async: let its /mfaVerify answer land.
    await act(async () => {
      verifyButton!.click();
      await Promise.resolve();
    });

    expect(fetchMock.mock.calls[0][0]).toBe("/api/mfaVerify");
    expect(useAuthStore.getState().authState).toBe("YES");
    expect(getCsrfToken()).toBe("mfa-session-token");

    await apiJson<unknown>("/exampleList", { method: "POST" });
    expect(fetchMock).toHaveBeenCalledTimes(2);
    const sentHeaders = new Headers(fetchMock.mock.calls[1][1]?.headers);
    expect(sentHeaders.get("X-CSRF-Token")).toBe("mfa-session-token");
  });
});
