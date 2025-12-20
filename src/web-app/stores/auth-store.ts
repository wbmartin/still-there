import { create } from "zustand";
import type {
  AuthState,
  AuthStateType,
  LoginRequest,
  LoginResponse,
  RefreshResponse,
} from "../types/auth-type";
import { apiRequest, setCsrfToken, clearCsrfToken } from "../utils/api";
import { clearAllCaches } from "../utils/cache-utils";

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
      set({
        authState: "NO",
        error: err instanceof Error ? err.message : "Unknown error",
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
      if (!response.ok) throw new Error("Refresh failed");
      const data = (await response.json()) as RefreshResponse;
      if (data.user) {
        // console.log("Auth refresh successful for user:", data.user);
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
      // console.log("Auth refresh not successful 2");
      set({
        authState: "NO",
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
