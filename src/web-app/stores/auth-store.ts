import { create } from "zustand";
import type {
  AuthState,
  AuthStateType,
  LoginRequest,
  LoginResponse,
  RefreshResponse,
} from "../types/auth-type";
import {
  apiRequest,
  setCsrfToken,
  clearCsrfToken,
  isApiUnreachable,
  setSessionExpiredListener,
} from "../utils/api";
import { clearAllCaches } from "../utils/cache-utils";
import { notifications } from "@mantine/notifications";

interface AuthStore extends Omit<AuthState, "csrfToken"> {
  mfaRequired: boolean;
  tempToken?: string;
  authenticateWithUsernamePassword: (
    userName: string,
    password: string,
  ) => Promise<void>;
  authenticateWithAuthToken: () => Promise<void>;
  logout: () => Promise<void>;
  changeSecurityContext: (newContext: number) => Promise<void>;
   changeOwnPassword: (
     oldPassword: string,
     newPassword: string,
   ) => Promise<void>;
}

export const useAuthStore = create<AuthStore>((set) => ({
  authState: "TBD" as AuthStateType,
  grantList: [],
  availableSecurityContextList: [],
  userName: "",
  securityContext: 0,
  postAuthDestination: "",
  loading: false,
  error: null,
  mfaRequired: false,
  tempToken: undefined,
  //============================^FUNCTION LINE ^============================
  authenticateWithUsernamePassword: async (
    userName: string,
    password: string,
  ) => {
    set({ loading: true, error: null });
    try {
      const loginRequest: LoginRequest = { userName, password };
      const response = await apiRequest("/auth", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(loginRequest),
      });
      if (!response.ok) throw new Error("Login failed");
      const data = (await response.json()) as LoginResponse;
      if (data.user) {
        if (data.user.tempToken) {
          // MFA required
          set({
            authState: "MFA",
            mfaRequired: true,
            tempToken: data.user.tempToken,
            userName: data.user.userName || "",
          });
        } else {
          setCsrfToken(data.csrfToken);
          set({
            authState: "YES",
            grantList: data.user.grantList || [],
            availableSecurityContextList:
              data.user.availableSecurityContextList || [],
            userName: data.user.userName || "",
            securityContext: data.user.securityContext || 0,
            mfaRequired: data.user.mfaRequired || false,
          });
        }
      }
    } catch (err) {
      // Deliberately still "NO" rather than "UNREACHABLE": the person is
      // standing at the login form and the useful thing is to leave them
      // there with an accurate message.
      set({
        authState: "NO",
        error: isApiUnreachable(err)
          ? "Could not reach the server. Check your connection and try again."
          : err instanceof Error
            ? err.message
            : "Unknown error",
      });
    } finally {
      set({ loading: false });
    }
  },
  //============================^FUNCTION LINE ^============================
  authenticateWithAuthToken: async () => {
    // console.log("refresh async starting ");
    set({ authState: "CHECKING", loading: true, error: null });
    try {
      const response = await apiRequest("/refresh", {});

      // Only an explicit refusal means signed out. A 502 from a proxy while
      // the API restarts is an ordinary HTTP response, not a verdict on the
      // session -- so the question is "did the server say this person is
      // not signed in", and only 401/403 say that.
      if (response.status === 401 || response.status === 403) {
        set({ authState: "NO", error: null });
        return;
      }
      if (!response.ok) {
        set({
          authState: "UNREACHABLE",
          error: `The server responded with ${response.status}`,
        });
        return;
      }

      const data = (await response.json()) as RefreshResponse;
      if (!data.user) {
        // A 200 with no user is a contract the server is not keeping.
        set({ authState: "NO", error: null });
        return;
      }
      setCsrfToken(data.csrfToken);
      set({
        authState: "YES",
        grantList: data.user.grantList || [],
        availableSecurityContextList:
          data.user.availableSecurityContextList || [],
        userName: data.user.userName || "",
        securityContext: data.user.securityContext || 0,
      });
    } catch (err) {
      // What reaches here is a connection failure or a body that would not
      // parse. Neither told us anything about the session, so neither may
      // claim it has ended.
      set({
        authState: "UNREACHABLE",
        error: err instanceof Error ? err.message : "Refresh failed",
      });
    } finally {
      set({ loading: false });
    }
  },
  //============================^FUNCTION LINE ^============================
  logout: async () => {
    try {
      await apiRequest("/logoff", { method: "POST" });
    } catch (err) {
      console.error(err);
    }
    clearCsrfToken();

    // Clear all application caches
    clearAllCaches();

    set({
      authState: "NO",
      grantList: [],
      availableSecurityContextList: [],
      userName: "",
      securityContext: 0,
      postAuthDestination: "",
      mfaRequired: false,
    });
  },

  changeSecurityContext: async (newContext: number) => {
    set({ loading: true, error: null });
    try {
      const response = await apiRequest("/changeSecurityContext", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ securityContext: newContext }),
      });
      if (!response.ok) throw new Error("Context change failed");
      const data = (await response.json()) as LoginResponse;
      if (data.user) {
        set({
          authState: "YES",
          grantList: data.user.grantList || [],
          availableSecurityContextList:
            data.user.availableSecurityContextList || [],
          userName: data.user.userName || "",
          securityContext: data.user.securityContext || 0,
        });
      }
    } catch (err) {
      set({
        error: err instanceof Error ? err.message : "Context change failed",
      });
    } finally {
      set({ loading: false });
    }
  },

  changeOwnPassword: async (oldPassword: string, newPassword: string) => {
    set({ loading: true, error: null });
    try {
      const response = await apiRequest("/changeOwnPassword", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ oldPassword, newPassword }),
      });
      if (!response.ok) {
        const errorData = (await response.json()) as { message: string };
        throw new Error(errorData.message || "Password change failed");
      }
    } catch (err) {
      set({
        error: err instanceof Error ? err.message : "Password change failed",
      });
      throw err; // Re-throw so the component can handle it
    } finally {
      set({ loading: false });
    }
  },
}));

// The API layer reports an ended session here rather than importing this
// store, which imports it -- a direct call would be a cycle.
//
// Deliberately NOT a call to logout(): the credential is already dead, so
// POST /logoff would fail. The local half of logout is what matters.
// postAuthDestination is left alone so the person lands back where they were.
setSessionExpiredListener(() => {
  // Several in-flight requests can each come back 401; the first one does
  // the work.
  if (useAuthStore.getState().authState === "NO") return;

  clearCsrfToken();
  clearAllCaches();
  const message = "Your session has ended. Please sign in again.";
  useAuthStore.setState({
    authState: "NO",
    grantList: [],
    availableSecurityContextList: [],
    userName: "",
    securityContext: 0,
    mfaRequired: false,
    error: message,
  });
  notifications.show({
    title: "Session ended",
    message,
    color: "orange",
    autoClose: 6000,
  });
});
