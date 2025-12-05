import { apiSvc } from "./api-svc";

interface AuthResponse {
  user: {
    userName: string;
    securityContext: number;
    grantList: string[];
  };
  csrfToken: string;
}

export interface AuthRequest {
  userName: string;
  password: string;
}

export const authApi = apiSvc.injectEndpoints({
  endpoints: (build) => ({
    login: build.mutation<AuthResponse, AuthRequest>({
      query: (body) => ({
        url: "/auth",
        method: "POST",
        body: {
          user_name: body.userName,
          password: body.password,
        },
      }),
      transformResponse: (response: any) => {
        const grantList = response.user?.user_grant_short_code_list ?
          Object.values(response.user.user_grant_short_code_list).flat() : [];
        return {
          user: response.user ? {
            userName: response.user.user_name,
            securityContext: response.user.security_context,
            grantList,
          } : undefined,
          csrfToken: response.csrf_token || '',
        };
      },
    }),
    refresh: build.query<AuthResponse, void>({
      query: () => "/refresh",
      transformResponse: (response: any) => {
        const grantList = response.user?.user_grant_short_code_list ?
          Object.values(response.user.user_grant_short_code_list).flat() : [];
        return {
          user: response.user ? {
            userName: response.user.user_name,
            securityContext: response.user.security_context,
            grantList,
          } : undefined,
          csrfToken: response.csrf_token || '',
        };
      },
    }),
    logout: build.query<AuthPackage, void>({
      query: () => ({
        url: "/logout",
        method: "GET",
      }),
    }),
  }),
});

export const { useLoginMutation, useLazyLogoutQuery, useRefreshQuery } = authApi;