export type AuthStateType = "YES" | "NO" | "CHECKING" | "TBD";

export interface User {
  userName: string;
  grantList: number[];
  availableSecurityContextList: number[];
  securityContext: number;
}

export interface LoginRequest {
  userName: string;
  password: string;
}

export interface LoginResponse {
  user: User;
  csrfToken: string;
}

export interface RefreshResponse {
  user: User;
  csrfToken: string;
}

export interface AuthState {
  authState: AuthStateType;
  grantList: number[];
  availableSecurityContextList: number[];
  userName: string;
  securityContext: number;
  postAuthDestination: string;
  loading: boolean;
  error: string | null;
}
