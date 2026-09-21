import { useState, useEffect } from "react";
import { APP_NAME } from "../constants/app";
import { useNavigate } from "react-router-dom";
import { useForm } from "@mantine/form";
import { zodResolver } from "mantine-form-zod-resolver";
import { loginSchema } from "../schemas/form-schemas";
import {
  TextInput,
  PasswordInput,
  Button,
  Paper,
  Title,
  Container,
  Text,
  Anchor,
  Center,
} from "@mantine/core";
import { useAuthStore } from "../stores/auth-store";
import { notifications } from "@mantine/notifications";
import { UserRegistrationModal } from "../components/user-registration-modal";
import { ROUTES } from "../constants/routes";

function Login() {
  const [registrationModalOpened, setRegistrationModalOpened] = useState(false);

  const form = useForm({
    initialValues: {
      username: '',
      password: '',
    },
    validate: zodResolver(loginSchema),
  });
   const {
     authenticateWithUsernamePassword,
     authState,
     postAuthDestination,
   } = useAuthStore();
   const navigate = useNavigate();

  useEffect(() => {
     if (authState === "YES") {
       // Navigate to postAuthDestination or dashboard
        const destination = postAuthDestination || ROUTES.DASHBOARD;
       void navigate(destination);
     }
   }, [authState, postAuthDestination, navigate]);

  //============================^ HOOK LINE ^====================

  const handleSubmit = form.onSubmit(async (values) => {
    await authenticateWithUsernamePassword(values.username, values.password);
    // username is userName
    const state = useAuthStore.getState();
    if (state.authState === "MFA") {
      // MFA required, navigate to setup
      void navigate(ROUTES.MFA_SETUP);
    } else if (state.authState !== "YES") {
      notifications.show({
        title: "Login Failed",
        message: state.error || "Invalid credentials",
        color: "red",
      });
    }
  });

  return (
    <Center h="100vh">
      <Container size="sm">
        <Paper shadow="md" p="xl">
          <Title order={2} mb="md" ta="center">
            Login to {APP_NAME}
          </Title>
          <form onSubmit={handleSubmit}>
            <TextInput
              label="Username"
              {...form.getInputProps('username')}
              mb="md"
            />
            <PasswordInput
              label="Password"
              {...form.getInputProps('password')}
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
    </Center>
  );
}

export default Login;
