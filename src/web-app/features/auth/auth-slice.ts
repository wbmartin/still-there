// Template from https://github.com/learnwithjason/lets-learn-redux-toolkit/blob/main/src/features/counter/counter-slice.ts

import type { RootState } from "../../store";
import { createSlice } from "@reduxjs/toolkit";
import { authApi } from "../../services/auth-svc";
import { notifications } from "@mantine/notifications";

interface AuthState {
  isAuthenticated: boolean;
  csrfToken: string;
  grantList: string[];
  username: string;
  securityContext: number;
}

const initialState: AuthState = {
  isAuthenticated: false,
  csrfToken: "",
  grantList: [],
  username: "",
  securityContext: 0,
};

const authSlice = createSlice({
  name: "auth",
  initialState,
  reducers: {
    logout: () => initialState,
  },
  extraReducers: (builder) => {
    builder
      .addMatcher(authApi.endpoints.login.matchFulfilled, (state, action) => {
        if (action.payload.loginSuccess) {
          state.isAuthenticated = action.payload.loginSuccess;
          state.csrfToken = action.payload.csrfToken;
          state.grantList = action.payload.user.grantList || [];
          state.username = action.payload.user.username || "";
          state.securityContext = action.payload.securityContext || 0;
          localStorage.setItem("csrfToken", action.payload.csrfToken);
        } else {
          state.isAuthenticated = false;
          state.grantList = [];
          notifications.show({
            title: "Login Failed",
            message: "Invalid credentials",
            color: "red",
          });
        }
      })
      .addMatcher(authApi.endpoints.logout.matchFulfilled, (state) => {
        state.isAuthenticated = false;
        state.csrfToken = "";
        state.grantList = [];
        state.username = "";
        state.securityContext = 0;
      });
  },
});

export const { logout } = authSlice.actions;
export default authSlice.reducer;
export const selectIsAuthenticated = (state: RootState) =>
  state.auth.isAuthenticated;
export const selectUserGrantList = (state: RootState) => state.auth.grantList;