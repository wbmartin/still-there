import { apiSvc } from "./api-svc";

interface AuthPackage {
  loginSuccess: boolean;
  csrfToken: string;
  securityContext: number;
  user: {
    username: string;
    grantList: string[];
  };
}

export interface AuthRequest {
  username: string;
  passwd: string;
}

export const authApi = apiSvc.injectEndpoints({
  endpoints: (build) => ({
    login: build.mutation<AuthPackage, AuthRequest>({
      query: (body) => ({
        url: "/auth",
        method: "POST",
        body,
      }),
    }),
    logout: build.query<AuthPackage, void>({
      query: () => ({
        url: "/logout",
        method: "GET",
      }),
    }),
  }),
});

export const { useLoginMutation, useLazyLogoutQuery } = authApi;