import { useState } from "react";
import { Button, Title, Group, Text, Select, Alert } from "@mantine/core";
import { useAuthStore } from "../stores/auth-store";
import useAuthGuard from "../hooks/use-auth-guard";

function UserPreferencesPage() {
  const authState = useAuthGuard();
  const {
    userName,
    securityContext,
    availableSecurityContextList: availableSecurityContexts,
    changeSecurityContext,
    loading,
    error,
  } = useAuthStore();
  const [selectedContext, setSelectedContext] = useState<string>(
    securityContext.toString(),
  );

  if (authState !== "YES") {
    return <div>Checking Auth...</div>;
  }

  const handleContextChange = async () => {
    const newContext = parseInt(selectedContext);
    if (newContext !== securityContext) {
      await changeSecurityContext(newContext);
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
              onClick={handleContextChange}
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

      {error && (
        <Alert color="red" mt="md">
          {error}
        </Alert>
      )}
    </div>
  );
}

export default UserPreferencesPage;
