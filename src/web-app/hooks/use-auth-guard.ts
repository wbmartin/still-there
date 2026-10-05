import { useEffect, useRef } from "react";
import { useNavigate, useLocation } from "react-router-dom";
import { useAuthStore } from "../stores/auth-store";
import { ROUTES } from "../constants/routes";
import { notifications } from "@mantine/notifications";
import { reportError } from "../utils/report-error";

const useAuthGuard = (requiredGrants: number[]) => {
  const { authState, grantList } = useAuthStore();
  const navigate = useNavigate();
  const location = useLocation();
  const handleAuthStateNoSingleFireComplete = useRef(false);
  const onLoginPage = location.pathname === ROUTES.LOGIN;

  useEffect(() => {
    if (authState === "TBD" && !onLoginPage) {
      // Trigger refresh - only once per component mount
      void useAuthStore
        .getState()
        .authenticateWithAuthToken()
        .catch((err) => reportError(err, "auth guard"));
    } else if (authState === "NO") {
      if (!handleAuthStateNoSingleFireComplete.current) {
        handleAuthStateNoSingleFireComplete.current = true;
        // Save destination and redirect to login
        const currentPath = location.pathname + location.search;
        useAuthStore.setState({ postAuthDestination: currentPath });
        void navigate(ROUTES.LOGIN);
      }
    } else if (authState === "YES") {
      // Check grants - user must have ALL required grants
      const hasAllGrants = requiredGrants.every((grantId) =>
        grantList.includes(grantId),
      );

      if (!hasAllGrants) {
        // Show notification about insufficient privileges
        notifications.show({
          title: "Access Denied",
          message:
            "You don't have sufficient privileges to access this page. Redirecting to dashboard.",
          color: "orange",
          autoClose: 4000,
        });
        // Redirect to dashboard
        void navigate(ROUTES.DASHBOARD);
      }
    }
    // "MFA": allow access to MFA setup.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [authState, grantList, requiredGrants]);

  // While the token refresh is in flight the caller shows a loader; the
  // login page never waits on it.
  if (authState === "TBD" && !onLoginPage) {
    return "CHECKING";
  }

  // Signed in without the page's grants: never "YES", so the page's data
  // effects (all gated on "YES") do not ask the API for what it will refuse
  // while the effect above sends the user to the dashboard.
  if (
    authState === "YES" &&
    !requiredGrants.every((grantId) => grantList.includes(grantId))
  ) {
    return "DENIED";
  }

  return authState;
};

export default useAuthGuard;
