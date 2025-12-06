import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { TextInput, PasswordInput, Button, Paper, Title, Container } from "@mantine/core";
import { useAuthStore } from "../stores/auth-store";
import { notifications } from "@mantine/notifications";

function Login() {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const { login, loading, error } = useAuthStore();
  const navigate = useNavigate();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    await login(username, password);
    if (useAuthStore.getState().isAuthenticated) {
      navigate("/dashboard");
    } else {
      notifications.show({
        title: "Login Failed",
        message: error || "Invalid credentials",
        color: "red",
      });
    }
  };

  return (
    <Container size="sm" mt="xl">
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
          <Button type="submit" fullWidth loading={loading}>
            Login
          </Button>
        </form>
      </Paper>
    </Container>
  );
}

export default Login;