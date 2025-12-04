import { createApi, fetchBaseQuery } from "@reduxjs/toolkit/query/react";

export const apiSvc = createApi({
  reducerPath: "apiSvc",
  baseQuery: fetchBaseQuery({
    baseUrl: "/api",
    credentials: "include", // Include cookies for JWT
  }),
  endpoints: () => ({}),
});

export default apiSvc;