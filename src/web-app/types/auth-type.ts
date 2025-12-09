export type AuthStateType = 'YES' | 'NO' | 'CHECKING' | 'TBD';

export interface User {
  userName: string;
  grantList: string[];
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
  grantList: string[];
  userName: string;
  securityContext: number;
  postAuthDestination: string;
  loading: boolean;
  error: string | null;
}