import { useEffect } from "react";
import { useNavigate, useLocation } from "react-router-dom";
import { useAuthStore } from "../stores/auth-store";
import { ROUTES } from "../constants/routes";

const useAuthGuard = () => {
  const { authState } = useAuthStore();
  const navigate = useNavigate();
  const location = useLocation();

  useEffect(() => {
    if (authState === "TBD") {
      // Trigger refresh
      void useAuthStore.getState().authenticateWithAuthToken();
    } else if (authState === "NO") {
      // Save destination
      const currentPath = location.pathname + location.search;
      useAuthStore.setState({ postAuthDestination: currentPath });
      void navigate(`/${ROUTES.LOGIN}`);
    }
  }, [authState, navigate, location]);

  return authState;
};

export default useAuthGuard;
