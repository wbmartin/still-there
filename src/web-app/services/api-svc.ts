import { createApi, fetchBaseQuery } from "@reduxjs/toolkit/query/react";
import { getCsrfToken } from "../utils/api";

export const apiSvc = createApi({
  reducerPath: "apiSvc",
  baseQuery: fetchBaseQuery({
    baseUrl: "/api",
    credentials: "include", // Include cookies for JWT
    prepareHeaders: (headers) => {
      const csrfToken = getCsrfToken();
      if (csrfToken) {
        headers.set("X-CSRF-Token", csrfToken);
      }
      return headers;
    },
  }),
  endpoints: () => ({}),
});

export default apiSvc;