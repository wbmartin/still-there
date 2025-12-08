import { useEffect } from "react";
import { useNavigate, useLocation } from "react-router-dom";
import { useAuthStore } from "../stores/auth-store";

const useAuthGuard = () => {
  const { authState } = useAuthStore();
  const navigate = useNavigate();
  const location = useLocation();

  useEffect(() => {
    if (authState === "TBD") {
      // Trigger refresh
      useAuthStore.getState().authenticateWithAuthToken();
    } else if (authState === "NO") {
      // Save destination
      const currentPath = location.pathname + location.search;
      useAuthStore.setState({ postAuthDestination: currentPath });
      navigate("/login");
    }
  }, [authState, navigate, location]);

  return authState;
};

export default useAuthGuard;
