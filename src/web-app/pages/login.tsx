import { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import {
  TextInput,
  PasswordInput,
  Button,
  Paper,
  Title,
  Container,
  Text,
  Anchor,
} from "@mantine/core";
import { useAuthStore } from "../stores/auth-store";
import { notifications } from "@mantine/notifications";
import { UserRegistrationModal } from "../components/UserRegistrationModal";
import { ROUTES } from "../constants/routes";

function Login() {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [registrationModalOpened, setRegistrationModalOpened] = useState(false);
   const {
     authenticateWithUsernamePassword,
     authState,
     postAuthDestination,
   } = useAuthStore();
   const navigate = useNavigate();

  useEffect(() => {
     if (authState === "YES") {
       // Navigate to postAuthDestination or dashboard
        const destination = postAuthDestination || `/${ROUTES.DASHBOARD}`;
       void navigate(destination);
     }
   }, [authState, postAuthDestination, navigate]);

  //============================^ HOOK LINE ^====================

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    await authenticateWithUsernamePassword(username, password);
    // username is userName
    const state = useAuthStore.getState();
    if (state.authState === "MFA") {
      // MFA required, navigate to setup
      void navigate(`/${ROUTES.MFA_SETUP}`);
    } else if (state.authState !== "YES") {
      notifications.show({
        title: "Login Failed",
        message: state.error || "Invalid credentials",
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
          <form onSubmit={(e) => void handleSubmit(e)}>
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
          <Text ta="center" mt="md">
            Don't have an account?{" "}
            <Anchor
              component="button"
              type="button"
              onClick={() => setRegistrationModalOpened(true)}
            >
              Create one here
            </Anchor>
          </Text>
        </Paper>
      </Container>
      <UserRegistrationModal
        opened={registrationModalOpened}
        onClose={() => setRegistrationModalOpened(false)}
      />
    </div>
  );
}

export default Login;
