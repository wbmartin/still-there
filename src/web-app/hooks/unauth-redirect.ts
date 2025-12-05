import { useAppSelector } from "../hooks";
import { useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { useRefreshQuery } from "../services/auth-svc";

const useUnauthRedirect = (postAuthTarget: string) => {
  const isAuthenticated = useAppSelector((state) => state.auth.isAuthenticated);
  const navigate = useNavigate();
  const { data, isLoading } = useRefreshQuery(undefined, { skip: isAuthenticated });

  useEffect(() => {
    if (!isAuthenticated && !isLoading) {
      if (data?.user) {
        // Refresh succeeded, state updated by slice
      } else {
        navigate("/login");
      }
    }
  }, [isAuthenticated, navigate, postAuthTarget, data, isLoading]);
};

export default useUnauthRedirect;