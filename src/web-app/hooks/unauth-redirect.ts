import { useAppSelector } from "../hooks";
import { useEffect } from "react";
import { useNavigate } from "react-router-dom";

const useUnauthRedirect = (postAuthTarget: string) => {
  const isAuthenticated = useAppSelector((state) => state.auth.isAuthenticated);
  const navigate = useNavigate();

  useEffect(() => {
    if (!isAuthenticated) {
      navigate("/login");
    }
  }, [isAuthenticated, navigate, postAuthTarget]);
};

export default useUnauthRedirect;