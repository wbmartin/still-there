import { create } from "zustand";
import type {
  AuthState,
  AuthStateType,
  LoginRequest,
  LoginResponse,
  RefreshResponse,
} from "../types/auth-type";
import { apiRequest, setCsrfToken, clearCsrfToken } from "../utils/api";

interface AuthStore extends Omit<AuthState, 'csrfToken'> {
  authenticateWithUsernamePassword: (
    userName: string,
    password: string,
  ) => Promise<void>;
  authenticateWithAuthToken: () => Promise<void>;
  logout: () => Promise<void>;
}

export const useAuthStore = create<AuthStore>((set) => ({
  authState: "TBD" as AuthStateType,
  grantList: [],
  userName: "",
  securityContext: 0,
  postAuthDestination: "",
  loading: false,
  error: null,
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
       const data = await response.json() as LoginResponse;
        if (data.user) {
          setCsrfToken(data.csrfToken);
          const grantMap = data.user.grantList || {};
          const allGrantIds: number[] = [];
          Object.values(grantMap).forEach(ids => allGrantIds.push(...ids));
          const uniqueGrantIds = [...new Set(allGrantIds)];
          set({
            authState: "YES",
            grantList: uniqueGrantIds,
            userName: data.user.userName || "",
            securityContext: data.user.securityContext || 0,
          });
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
       const data = await response.json() as RefreshResponse;
        if (data.user) {
          // console.log("Auth refresh successful for user:", data.user);
          setCsrfToken(data.csrfToken);
          const grantMap = data.user.grantList || {};
          const allGrantIds: number[] = [];
          Object.values(grantMap).forEach(ids => allGrantIds.push(...ids));
          const uniqueGrantIds = [...new Set(allGrantIds)];
          set({
            authState: "YES",
            grantList: uniqueGrantIds,
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
    set({
      authState: "NO",
      grantList: [],
      userName: "",
      securityContext: 0,
      postAuthDestination: "",
    });
  },
}));
