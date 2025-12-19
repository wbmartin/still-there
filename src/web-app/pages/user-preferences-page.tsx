import { useState } from "react";
import { Button, Title, Group, Text, Select, Alert } from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { useNavigate } from "react-router-dom";
import { useAuthStore } from "../stores/auth-store";
import { useExamplesStore } from "../stores/examples-store";
import { useAppUserStore } from "../stores/app-user-store";
import { useAppUserGroupStore } from "../stores/app-user-group-store";
import { ROUTES } from "../constants/routes";
import useAuthGuard from "../hooks/use-auth-guard";
import { grants } from "../constants/grants";
import { ChangePasswordModal } from "../components/ChangePasswordModal";

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
  const [passwordChangeError, setPasswordChangeError] = useState<string | undefined>();

  const handleClearCache = () => {
    useExamplesStore.getState().clearCache();
    useAppUserStore.getState().clearCache();
    useAppUserGroupStore.getState().clearCache();
    notifications.show({
      title: "Cache Cleared",
      message: "All cached data has been cleared. Data will be refetched on next access.",
      color: "green",
    });
  };

  if (authState !== "YES") {
    return <div>Checking Auth...</div>;
  }

  const handleContextChange = async () => {
    const newContext = parseInt(selectedContext);
    if (newContext !== securityContext) {
      await changeSecurityContext(newContext);
      // Clear all caches after successful context change
      useExamplesStore.getState().clearCache();
      useAppUserStore.getState().clearCache();
      useAppUserGroupStore.getState().clearCache();
      // Navigate to dashboard
      void navigate(`/${ROUTES.DASHBOARD}`);
    }
  };

  const handlePasswordChange = async (oldPassword: string, newPassword: string) => {
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
      setPasswordChangeError(err instanceof Error ? err.message : "Failed to change password");
    } finally {
      setPasswordChangeLoading(false);
    }
  };

  const contextOptions = availableSecurityContexts.map((ctx) => ({
    value: ctx.toString(),
    label: `Context ${ctx}`,
  }));

  return (
    <div style={{ padding: "2rem", maxWidth: "600px" }}>
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
        <div>
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
              style={{ minWidth: "200px" }}
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
        </div>
      )}

      {availableSecurityContexts.length <= 1 && (
        <Text c="dimmed">No additional security contexts available.</Text>
      )}

      <div style={{ marginTop: "2rem" }}>
        <Title order={3} mb="sm">
          Password Management
        </Title>
        <Group mb="md">
          <Button onClick={() => setPasswordModalOpen(true)} variant="outline">
            Change Password
          </Button>
        </Group>
        <Text size="sm" c="dimmed">
          Change your account password. You will need to enter your current password.
        </Text>
      </div>

      <div style={{ marginTop: "2rem" }}>
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
      </div>

      {error && (
        <Alert color="red" mt="md">
          {error}
        </Alert>
      )}

      <ChangePasswordModal
        isOpen={passwordModalOpen}
        onConfirm={handlePasswordChange}
        onCancel={() => {
          setPasswordModalOpen(false);
          setPasswordChangeError(undefined);
        }}
        mode="own"
        loading={passwordChangeLoading}
        error={passwordChangeError}
      />
    </div>
  );
}

export default UserPreferencesPage;
