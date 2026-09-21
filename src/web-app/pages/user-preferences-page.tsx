import { useState } from "react";
import { Button, Title, Group, Text, Select, Alert, Container, Stack, Loader } from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { useNavigate } from "react-router-dom";
import { useAuthStore } from "../stores/auth-store";
import { clearAllCaches } from "../utils/cache-utils";
import { ROUTES } from "../constants/routes";
import useAuthGuard from "../hooks/use-auth-guard";
import { grants } from "../constants/grants";
import { ChangePasswordModal } from "../components/change-password-modal";

function UserPreferencesPage() {
  const authState = useAuthGuard([grants.LOGON]);
  const navigate = useNavigate();
  const {
    userName,
    securityContext,
    availableSecurityContextList: availableSecurityContexts,
    changeSecurityContext,
    changeOwnPassword,
    loading,
    error,
  } = useAuthStore();
  const [selectedContext, setSelectedContext] = useState<string>(
    securityContext.toString(),
  );
  const [passwordModalOpen, setPasswordModalOpen] = useState(false);
  const [passwordChangeLoading, setPasswordChangeLoading] = useState(false);
  const [passwordChangeError, setPasswordChangeError] = useState<
    string | undefined
  >();

  const handleClearCache = () => {
    clearAllCaches();
    notifications.show({
      title: "Cache Cleared",
      message:
        "All cached data has been cleared. Data will be refetched on next access.",
      color: "green",
    });
  };

  //============================^ HOOK LINE ^====================
  if (authState === "CHECKING") {
    return <Container size={1280} px={32} py={32} mx="auto" maw="600px" ta="center"><Loader /></Container>;
  }
  if (authState === "NO" || authState === "TBD") {
    return <Container size={1280} px={32} py={32} mx="auto" maw="600px"><Alert color="red">Authentication required.</Alert></Container>;
  }

  const handleContextChange = async () => {
    const newContext = parseInt(selectedContext);
    if (newContext !== securityContext) {
      await changeSecurityContext(newContext);
      // Clear all caches after successful context change
      clearAllCaches();
      // Navigate to dashboard
      void navigate(ROUTES.DASHBOARD);
    }
  };

  const handlePasswordChange = async (
    oldPassword: string,
    newPassword: string,
  ) => {
    setPasswordChangeLoading(true);
    setPasswordChangeError(undefined);
    try {
      await changeOwnPassword(oldPassword, newPassword);
      setPasswordModalOpen(false);
      notifications.show({
        title: "Success",
        message: "Password changed successfully",
        color: "green",
      });
    } catch (err) {
      setPasswordChangeError(
        err instanceof Error ? err.message : "Failed to change password",
      );
    } finally {
      setPasswordChangeLoading(false);
    }
  };

  const contextOptions = availableSecurityContexts.map((ctx) => ({
    value: ctx.toString(),
    label: `Context ${ctx}`,
  }));

  return (
    <Container size={1280} px={32} py={32} mx="auto" maw="600px">
      <Title order={1} mb="lg">
        User Preferences
      </Title>

      <Group mb="md">
        <Text fw={500}>Username:</Text>
        <Text>{userName}</Text>
      </Group>

      <Group mb="md">
        <Text fw={500}>Current Security Context:</Text>
        <Text>{securityContext}</Text>
      </Group>

      {availableSecurityContexts.length > 1 && (
        <Stack>
          <Title order={3} mb="sm">
            Change Security Context
          </Title>
          <Group mb="md">
            <Select
              data={contextOptions}
              value={selectedContext}
              onChange={(value) =>
                setSelectedContext(value || securityContext.toString())
              }
              placeholder="Select context"
              miw="200px"
            />
            <Button
              onClick={() => void handleContextChange()}
              loading={loading}
              disabled={parseInt(selectedContext) === securityContext}
            >
              Change Context
            </Button>
          </Group>
          <Text size="sm" c="dimmed">
             Changing context will update your permissions and available
             features.
         </Text>
       </Stack>
      )}

      {availableSecurityContexts.length <= 1 && (
        <Text c="dimmed">No additional security contexts available.</Text>
      )}

      <Stack mt="2rem">
        <Title order={3} mb="sm">
          Password Management
        </Title>
        <Group mb="md">
          <Button onClick={() => setPasswordModalOpen(true)} variant="outline">
            Change Password
          </Button>
        </Group>
        <Text size="sm" c="dimmed">
          Change your account password. You will need to enter your current
          password.
        </Text>
      </Stack>

      <Stack mt="2rem">
        <Title order={3} mb="sm">
          Administration
        </Title>
        <Group mb="md">
          <Button
            onClick={() => void navigate(ROUTES.APP_USER_LIST)}
            variant="outline"
            disabled={
              !useAuthStore.getState().grantList.includes(grants.CREATE_USER)
            }
          >
            User Admin
          </Button>
          <Button
            onClick={() => void navigate(ROUTES.APP_USER_GROUP_LIST)}
            variant="outline"
            disabled={
              !useAuthStore
                .getState()
                .grantList.includes(grants.CREATE_USER_GROUP)
            }
          >
            User Groups
          </Button>
        </Group>
        <Text size="sm" c="dimmed">
          Manage users and user groups (requires appropriate permissions).
        </Text>
      </Stack>

      <Stack mt="2rem">
        <Title order={3} mb="sm">
          Account Management
        </Title>
        <Group mb="md">
          <Button
            onClick={() => {
              void useAuthStore
                .getState()
                .logout()
                .then(() => navigate(ROUTES.LOGIN));
            }}
            variant="outline"
            color="red"
          >
            Logout
          </Button>
        </Group>
        <Text size="sm" c="dimmed">
          Log out of your account. You will need to log in again to access
          the application.
        </Text>
      </Stack>

      <Stack mt="2rem">
        <Title order={3} mb="sm">
          Cache Management
        </Title>
        <Group mb="md">
          <Button onClick={() => void handleClearCache()} variant="outline">
            Clear Cache
          </Button>
        </Group>
        <Text size="sm" c="dimmed">
          Clear all cached data. This will force fresh data to be loaded on next
          access.
        </Text>
      </Stack>

      {error && (
        <Alert color="red" mt="md">
          {error}
        </Alert>
      )}

      <ChangePasswordModal
        opened={passwordModalOpen}
        onSubmit={(oldPassword, newPassword) => {
          void handlePasswordChange(oldPassword, newPassword);
        }}
        onClose={() => {
          setPasswordModalOpen(false);
          setPasswordChangeError(undefined);
        }}
        mode="own"
        loading={passwordChangeLoading}
        error={passwordChangeError}
      />
    </Container>
  );
}

export default UserPreferencesPage;
