import { create } from "zustand";
import type {
  AuthState,
  LoginRequest,
  LoginResponse,
  RefreshResponse,
} from "../types/auth-type";
import {
  apiJson,
  apiRequest,
  jsonBody,
  setCsrfToken,
  clearCsrfToken,
  isApiUnreachable,
  setSessionResyncListener,
  subscribeApiOutcome,
} from "../utils/api";
import { clearAllCaches } from "../utils/cache-utils";
import { API_ENDPOINTS } from "../constants/api";
import { ROUTES } from "../constants/routes";
import { notifications } from "@mantine/notifications";
import { reportError } from "../utils/report-error";

/**
 * How long the page-load /refresh keeps asking while the API is unreachable
 * (a failed fetch, 5xx or 429): three more attempts, 1 s / 2 s / 4 s apart,
 * so a page loaded during an API restart waits it out instead of showing
 * the outage notice. Then authState is "UNREACHABLE".
 */
const REFRESH_RETRY_DELAY_MS = [1000, 2000, 4000];

const wait = (ms: number): Promise<void> =>
  new Promise((resolve) => setTimeout(resolve, ms));

const requestRefreshWithRetry = async (): Promise<Response> => {
  for (let attempt = 0; ; attempt++) {
    try {
      return await apiRequest(API_ENDPOINTS.REFRESH, {});
    } catch (err) {
      if (!isApiUnreachable(err)) throw err;
      if (attempt >= REFRESH_RETRY_DELAY_MS.length) throw err;
      await wait(REFRESH_RETRY_DELAY_MS[attempt]);
    }
  }
};

interface AuthStore extends Omit<AuthState, "csrfToken"> {
  mfaRequired: boolean;
  tempToken?: string;
  /** /auth's answer for this temp token: enrolled users verify, the rest enrol. */
  mfaEnrolled: boolean;
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
  authState: "TBD",
  grantList: [],
  availableSecurityContextList: [],
  userName: "",
  securityContext: 0,
  postAuthDestination: "",
  loading: false,
  error: null,
  mfaRequired: false,
  tempToken: undefined,
  mfaEnrolled: false,
  //============================^FUNCTION LINE ^============================
  authenticateWithUsernamePassword: async (
    userName: string,
    password: string,
  ) => {
    set({ loading: true, error: null });
    try {
      const loginRequest: LoginRequest = { userName, password };
      // anonymous: a 401 here is "wrong password", not an ended session
      const data = await apiJson<LoginResponse>(
        API_ENDPOINTS.AUTH,
        { method: "POST", ...jsonBody(loginRequest) },
        { anonymous: true },
      );
      if (data.user) {
        if (data.user.tempToken) {
          // MFA required
          set({
            authState: "MFA",
            mfaRequired: true,
            tempToken: data.user.tempToken,
            mfaEnrolled: data.user.mfaEnrolled === true,
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
      // there with an accurate message. A 5xx or 429 carries the server's
      // own words (an account lockout is a 429); a failed fetch has none.
      set({
        authState: "NO",
        error: isApiUnreachable(err) && err.status === undefined
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
    set({ authState: "CHECKING", loading: true, error: null });
    try {
      const response = await requestRefreshWithRetry();

      // Only an explicit refusal means signed out. A 502 from a proxy while
      // the API restarts (apiRequest throws it as ApiUnreachableError, and
      // it is retried above) is not a verdict on the session -- so the
      // question is "did the server say this person is not signed in", and
      // only 401/403 say that.
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
      // What reaches here is an outage that outlasted the retries or a body
      // that would not parse. Neither told us anything about the session,
      // so neither may claim it has ended.
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
      await apiRequest(API_ENDPOINTS.LOGOFF, { method: "POST" });
    } catch (err) {
      reportError(err, "logout");
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
      mfaEnrolled: false,
    });
  },

  changeSecurityContext: async (newContext: number) => {
    set({ loading: true, error: null });
    try {
      const data = await apiJson<LoginResponse>(
        API_ENDPOINTS.CHANGE_SECURITY_CONTEXT,
        { method: "POST", ...jsonBody({ securityContext: newContext }) },
      );
      if (data.user) {
        // A new session id, so a new CSRF token: without it this tab's next
        // write is refused.
        setCsrfToken(data.csrfToken);
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
      await apiJson<unknown>(API_ENDPOINTS.CHANGE_OWN_PASSWORD, {
        method: "POST",
        ...jsonBody({ oldPassword, newPassword }),
      });
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

// The API layer announces an ended session through subscribeApiOutcome
// rather than importing this store, which imports it -- a direct call would
// be a cycle.
//
// Deliberately NOT a call to logout(): the credential is already dead, so
// POST /logoff would fail. The local half of logout is what matters.
// postAuthDestination is left alone so the person lands back where they were.
subscribeApiOutcome((outcome) => {
  if (outcome.kind !== "session-expired") return;
  // Only an established session can end. Several in-flight requests can
  // each come back 401 (the first does the work), and a 401 while CHECKING
  // or at the login page is someone who was never signed in: no toast.
  if (useAuthStore.getState().authState !== "YES") return;

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
    mfaEnrolled: false,
    error: message,
  });
  notifications.show({
    title: "Session ended",
    message,
    color: "orange",
    autoClose: 6000,
  });
});

// The store cannot reach the router; the app registers how to navigate.
let routeNavigator: ((path: string) => void) | null = null;

export const setRouteNavigator = (
  navigator: ((path: string) => void) | null,
): void => {
  routeNavigator = navigator;
};

// A write was refused because another tab moved the shared cookie. Same user
// and tenant: the API layer takes the new token and resends. Anyone else: the
// write is dropped and this tab reloads into the identity the cookie holds,
// since what is on screen was authored under one that no longer has it.
setSessionResyncListener({
  isSameIdentity: (refreshed) => {
    const state = useAuthStore.getState();
    return (
      refreshed.user.userName === state.userName &&
      refreshed.user.securityContext === state.securityContext
    );
  },
  adoptSession: (refreshed) => {
    setCsrfToken(refreshed.csrfToken);
    clearAllCaches();
    useAuthStore.setState({
      authState: "YES",
      grantList: refreshed.user.grantList || [],
      availableSecurityContextList:
        refreshed.user.availableSecurityContextList || [],
      userName: refreshed.user.userName || "",
      securityContext: refreshed.user.securityContext || 0,
      postAuthDestination: "",
      mfaRequired: refreshed.user.mfaRequired || false,
      mfaEnrolled: false,
      error: null,
    });
    notifications.show({
      title: "Session changed",
      message:
        "Your session changed in another tab. Unsaved changes here were " +
        "not saved.",
      color: "orange",
      autoClose: 6000,
    });
    routeNavigator?.(ROUTES.DASHBOARD);
  },
});
