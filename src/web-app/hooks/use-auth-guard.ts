import { useEffect, useState } from "react";
import { useNavigate, useLocation } from "react-router-dom";
import { useAuthStore } from "../stores/auth-store";
import { ROUTES } from "../constants/routes";
import { notifications } from "@mantine/notifications";

const useAuthGuard = (requiredGrants: number[]) => {
  const { authState, grantList } = useAuthStore();
  const navigate = useNavigate();
  const location = useLocation();
  const [isValidating, setIsValidating] = useState(false);

  useEffect(() => {
    if (authState === "TBD") {
      setIsValidating(true);
      // Trigger refresh - only once per component mount
      void useAuthStore.getState().authenticateWithAuthToken();
    } else if (authState === "NO") {
      setIsValidating(false);
      // Save destination and redirect to login
      const currentPath = location.pathname + location.search;
      useAuthStore.setState({ postAuthDestination: currentPath });
      navigate(`/${ROUTES.LOGIN}`);
    } else if (authState === "YES") {
      // Check grants - user must have ALL required grants
      const hasAllGrants = requiredGrants.every(grantId =>
        grantList.includes(grantId)
      );

      if (!hasAllGrants) {
        setIsValidating(false);
        // Show notification about insufficient privileges
        notifications.show({
          title: "Access Denied",
          message: "You don't have sufficient privileges to access this page. Redirecting to dashboard.",
          color: "orange",
          autoClose: 4000,
        });
        // Redirect to dashboard
        navigate(`/${ROUTES.DASHBOARD}`);
      } else {
        setIsValidating(false);
      }
    } else {
      setIsValidating(false);
    }
  }, [authState]); // Only depend on authState to prevent loops

  // Return auth state with loading indicator
  if (isValidating) {
    return "CHECKING";
  }

  return authState;
};

export default useAuthGuard;
