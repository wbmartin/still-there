import { useLocation } from "react-router-dom";
import { ROUTES } from "../constants/routes";

// The help key of the page shown: its ROUTES value, the path without a
// query string or trailing slash. The index route is the login form.
export const routeKeyOf = (pathname: string): string => {
  const path = pathname.replace(/\/+$/, "");
  return path === "" ? ROUTES.LOGIN : path;
};

export function useRouteKey(): string {
  return routeKeyOf(useLocation().pathname);
}
