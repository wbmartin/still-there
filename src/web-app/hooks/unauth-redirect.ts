import { useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { useAuthStore } from "../stores/auth-store";

const useUnauthRedirect = (postAuthTarget: string) => {
  const { isAuthenticated, refresh } = useAuthStore();
  const navigate = useNavigate();

  useEffect(() => {
    if (!isAuthenticated) {
      refresh().then(() => {
        if (!useAuthStore.getState().isAuthenticated) {
          navigate("/login");
        }
      });
    }
  }, [isAuthenticated, navigate, postAuthTarget, refresh]);
};

export default useUnauthRedirect;