/**
 * "NO" means the server said this person is not signed in.
 * "UNREACHABLE" means the server did not answer at all, which is a different
 * fact and deserves a different screen -- see ApiUnreachableError in utils/api.
 * Collapsing the two is what makes a down API look like an expired session.
 */
export type AuthStateType =
  | "YES"
  | "NO"
  | "CHECKING"
  | "TBD"
  | "MFA"
  | "UNREACHABLE";

export interface User {
  userName: string;
  grantList: number[];
  availableSecurityContextList: number[];
  securityContext: number;
  mfaRequired?: boolean;
  tempToken?: string;
  /** With a temp token: verify a code (true) or enrol (false). */
  mfaEnrolled?: boolean;
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
