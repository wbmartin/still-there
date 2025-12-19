import { useEffect } from "react";
import { useNavigate, useLocation } from "react-router-dom";
import { useAuthStore } from "../stores/auth-store";
import { ROUTES } from "../constants/routes";

const useAuthGuard = (requiredGrants: number[]) => {
  const { authState, grantList } = useAuthStore();
  const navigate = useNavigate();
  const location = useLocation();

  useEffect(() => {
    if (authState === "TBD") {
      // Trigger refresh - only once per component mount
      void useAuthStore.getState().authenticateWithAuthToken();
    } else if (authState === "NO") {
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
        // Redirect to dashboard without error message
        navigate(`/${ROUTES.DASHBOARD}`);
      }
    }
  }, [authState]); // Only depend on authState to prevent loops

  return authState;
};

export default useAuthGuard;
