export type AuthStateType = 'YES' | 'NO' | 'CHECKING' | 'TBD';

export interface User {
  userName: string;
  grantList: string[];
  securityContext: number;
}

export interface LoginRequest {
  user_name: string;
  password: string;
}

export interface LoginResponse {
  user: User;
  csrf_token: string;
}

export interface RefreshResponse {
  user: User;
  csrf_token: string;
}

export interface AuthState {
  authState: AuthStateType;
  grantList: string[];
  username: string;
  securityContext: number;
  postAuthDestination: string;
  loading: boolean;
  error: string | null;
}