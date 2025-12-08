import { create } from "zustand";
import type {
  AuthState,
  AuthStateType,
  LoginRequest,
  LoginResponse,
  RefreshResponse,
} from "../types/auth-type";
import { apiRequest } from "../utils/api";

interface AuthStore extends AuthState {
  authenticateWithUsernamePassword: (
    username: string,
    password: string,
  ) => Promise<void>;
  authenticateWithAuthToken: () => Promise<void>;
  logout: () => Promise<void>;
}

export const useAuthStore = create<AuthStore>((set, get) => ({
  authState: "TBD" as AuthStateType,
  csrfToken: "",
  grantList: [],
  username: "",
  securityContext: 0,
  postAuthDestination: "",
  loading: false,
  error: null,
  authenticateWithUsernamePassword: async (
    username: string,
    password: string,
  ) => {
    set({ loading: true, error: null });
    try {
      const loginRequest: LoginRequest = { user_name: username, password };
      const response = await apiRequest("/auth", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(loginRequest),
      });
      if (!response.ok) throw new Error("Login failed");
      const data: LoginResponse = await response.json();
      if (data.user) {
        localStorage.setItem('csrfToken', data.csrf_token);
        set({
          authState: "YES",
          csrfToken: data.csrf_token,
          grantList: data.user.grantList || [],
          username: data.user.userName || "",
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
  authenticateWithAuthToken: async () => {
    // console.log("refresh async starting ");
    set({ authState: "CHECKING", loading: true, error: null });
    try {
      const response = await apiRequest("/refresh", {}, get().csrfToken);
      if (!response.ok) throw new Error("Refresh failed");
      const data: RefreshResponse = await response.json();
      if (data.user) {
        // console.log("Auth refresh successful for user:", data.user);
        localStorage.setItem('csrfToken', data.csrf_token);
        set({
          authState: "YES",
          csrfToken: data.csrf_token,
          grantList: data.user.grantList || [],
          username: data.user.userName || "",
          securityContext: data.user.securityContext || 0,
        });
      } else {
        // console.log("Auth refresh not successful 1 ");
        set({ authState: "NO" });
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
  logout: async () => {
    try {
      await apiRequest("/logoff", { method: "POST" }, get().csrfToken);
    } catch (err) {
      console.error(err);
    }
    localStorage.removeItem('csrfToken');
    set({
      authState: "NO",
      csrfToken: "",
      grantList: [],
      username: "",
      securityContext: 0,
      postAuthDestination: "",
    });
  },
}));
