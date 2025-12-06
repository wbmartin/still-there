// Template from https://github.com/learnwithjason/lets-learn-redux-toolkit/blob/main/src/app/store.ts
import { configureStore } from "@reduxjs/toolkit";
import apiSvc from "./services/api-svc";
import authReducer from "./features/auth/auth-slice";

export const store = configureStore({
  reducer: {
    auth: authReducer,
    [apiSvc.reducerPath]: apiSvc.reducer,
  },
  middleware: (getDefaultMiddleware) => {
    return getDefaultMiddleware({
      immutableCheck: false,
      serializableCheck: false,
    }).concat(apiSvc.middleware);
  },
});

export type AppDispatch = typeof store.dispatch;
export type RootState = ReturnType<typeof store.getState>;
export const getState = () => store.getState;