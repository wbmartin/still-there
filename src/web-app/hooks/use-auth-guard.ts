import { useEffect, useState, useRef } from "react";
import { useNavigate, useLocation } from "react-router-dom";
import { useAuthStore } from "../stores/auth-store";
import { ROUTES } from "../constants/routes";
import { notifications } from "@mantine/notifications";
import { reportError } from "../utils/report-error";

const useAuthGuard = (requiredGrants: number[]) => {
  const { authState, grantList } = useAuthStore();
  const navigate = useNavigate();
  const location = useLocation();
  const [isValidating, setIsValidating] = useState(false);
  const handleAuthStateNoSingleFireComplete = useRef(false);

  useEffect(() => {
    if (authState === "TBD" && location.pathname !== `/${ROUTES.LOGIN}`) {
      setTimeout(() => setIsValidating(true), 0);
      // Trigger refresh - only once per component mount
      void useAuthStore
        .getState()
        .authenticateWithAuthToken()
        .catch((err) => reportError(err, "auth guard"));
    } else if (authState === "NO") {
      if (!handleAuthStateNoSingleFireComplete.current) {
        handleAuthStateNoSingleFireComplete.current = true;
        setIsValidating(false);
        // Save destination and redirect to login
        const currentPath = location.pathname + location.search;
        useAuthStore.setState({ postAuthDestination: currentPath });
        void navigate(`/${ROUTES.LOGIN}`);
      }
    } else if (authState === "MFA") {
      setIsValidating(false);
      // Allow access to MFA setup
    } else if (authState === "YES") {
      // Check grants - user must have ALL required grants
      const hasAllGrants = requiredGrants.every((grantId) =>
        grantList.includes(grantId),
      );

      if (!hasAllGrants) {
        setIsValidating(false);
        // Show notification about insufficient privileges
        notifications.show({
          title: "Access Denied",
          message:
            "You don't have sufficient privileges to access this page. Redirecting to dashboard.",
          color: "orange",
          autoClose: 4000,
        });
        // Redirect to dashboard
        void navigate(`/${ROUTES.DASHBOARD}`);
      } else {
        setIsValidating(false);
      }
    } else {
      setIsValidating(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [authState, grantList, requiredGrants]);

  // Return auth state with loading indicator
  if (isValidating) {
    return "CHECKING";
  }

  return authState;
};

export default useAuthGuard;
