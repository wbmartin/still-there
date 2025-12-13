import { useState, useEffect, useRef } from "react";
import { useNavigate } from "react-router-dom";
import {
  TextInput,
  PasswordInput,
  Button,
  Paper,
  Title,
  Container,
} from "@mantine/core";
import { useAuthStore } from "../stores/auth-store";
import { notifications } from "@mantine/notifications";

function Login() {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const {
    authenticateWithUsernamePassword,
    authState,
    postAuthDestination,
    error,
  } = useAuthStore();
  const navigate = useNavigate();
  const refreshCalled = useRef(false);

  useEffect(() => {
    if (authState === "TBD" && !refreshCalled.current) {
      refreshCalled.current = true;
      // Try refresh
      useAuthStore.getState().authenticateWithAuthToken();
    } else if (authState === "YES") {
      // Navigate to postAuthDestination or dashboard
      const destination = postAuthDestination || "/dashboard";
      navigate(destination);
    }
  }, [authState, postAuthDestination, navigate]);

  //============================^HOOOK LINE ^====================

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    await authenticateWithUsernamePassword(username, password);
    // username is userName
    const state = useAuthStore.getState();
    if (state.authState !== "YES") {
      notifications.show({
        title: "Login Failed",
        message: error || "Invalid credentials",
        color: "red",
      });
    }
  };

  return (
    <div
      style={{
        position: "absolute",
        top: "50%",
        left: "50%",
        transform: "translate(-50%, -50%)",
      }}
    >
      <Container size="sm">
        <Paper shadow="md" p="xl">
          <Title order={2} mb="md" ta="center">
            Login to Proto
          </Title>
          <form onSubmit={handleSubmit}>
            <TextInput
              label="Username"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              required
              mb="md"
            />
            <PasswordInput
              label="Password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
              mb="md"
            />
            <Button type="submit" fullWidth loading={authState === "CHECKING"}>
              Login
            </Button>
          </form>
        </Paper>
      </Container>
    </div>
  );
}

export default Login;
