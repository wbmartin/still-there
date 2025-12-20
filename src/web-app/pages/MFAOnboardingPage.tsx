import { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import {
  Container,
  Paper,
  Title,
  Text,
  TextInput,
  Button,
  Group,
  CopyButton,
  ActionIcon,
  Tooltip,
  Image,
} from "@mantine/core";
import { IconCopy, IconCheck } from "@tabler/icons-react";
import { notifications } from "@mantine/notifications";
import { apiRequest } from "../utils/api";
import { API_ENDPOINTS } from "../constants/api";
import { useAuthStore } from "../stores/auth-store";
import { LoginResponse } from "../types/auth-type";

function MFAOnboardingPage() {
  const [secret, setSecret] = useState("");
  const [qrCodeImage, setQrCodeImage] = useState("");
  const [totpCode, setTotpCode] = useState("");
  const [loading, setLoading] = useState(false);
  const [verifying, setVerifying] = useState(false);
  const navigate = useNavigate();
  const { tempToken } = useAuthStore();
  const [hasFetched, setHasFetched] = useState(false);

  useEffect(() => {
    if (!tempToken || hasFetched) return;

    setHasFetched(true);

    const fetchSetup = async () => {
      setLoading(true);
      try {
        const response = await apiRequest(API_ENDPOINTS.MFA_SETUP, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ tempToken }),
        });
        if (!response.ok) {
          throw new Error("Failed to get MFA setup data");
        }
        const data = (await response.json()) as {
          data: { secret: string; qrCodeImage: string };
        };
        setSecret(data.data.secret);
        setQrCodeImage(data.data.qrCodeImage);
      } catch (err) {
        notifications.show({
          title: "Error",
          message: err instanceof Error ? err.message : "Unknown error",
          color: "red",
        });
      } finally {
        setLoading(false);
      }
    };

    void fetchSetup();
  }, [tempToken, hasFetched]);

  // ^^^^^^^^^^^^^^^^^^^^^^^^ Hook Line  ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

  const handleVerify = async () => {
    setVerifying(true);
    try {
      const response = await apiRequest(API_ENDPOINTS.MFA_VERIFY, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ tempToken, totpCode }),
      });
      if (!response.ok) {
        throw new Error("MFA verification failed");
      }
       // On success, update auth state and navigate
       const data = (await response.json()) as LoginResponse;
       if (data.user) {
         // Update auth store with user data
         useAuthStore.setState({
           authState: "YES",
           grantList: data.user.grantList || [],
           availableSecurityContextList:
             data.user.availableSecurityContextList || [],
           userName: data.user.userName || "",
           securityContext: data.user.securityContext || 0,
           mfaRequired: data.user.mfaRequired || false,
         });
       }
       notifications.show({
         title: "Success",
         message: "MFA setup completed successfully",
         color: "green",
       });
       void navigate("/dashboard");
    } catch (err) {
      notifications.show({
        title: "Verification Failed",
        message: err instanceof Error ? err.message : "Invalid TOTP code",
        color: "red",
      });
    } finally {
      setVerifying(false);
    }
  };

  if (loading) {
    return (
      <Container size="sm">
        <Paper shadow="md" p="xl" ta="center">
          <Title order={2} mb="md">
            Setting up MFA
          </Title>
          <Text>Loading...</Text>
        </Paper>
      </Container>
    );
  }

  return (
    <Container size="sm">
      <Paper shadow="md" p="xl">
        <Title order={2} mb="md" ta="center">
          Set up Multi-Factor Authentication
        </Title>
        <Text mb="md">
          Scan the QR code with your authenticator app, or manually enter the
          secret key.
        </Text>

        {qrCodeImage && (
          <Group justify="center" mb="md">
            <Image src={qrCodeImage} alt="QR Code" width={100} fit="contain" />
          </Group>
        )}

        {secret && (
          <Group mb="md">
            <Text size="sm">Secret Key: {secret}</Text>
            <CopyButton value={secret}>
              {({ copied, copy }) => (
                <Tooltip label={copied ? "Copied" : "Copy"}>
                  <ActionIcon color={copied ? "teal" : "blue"} onClick={copy}>
                    {copied ? <IconCheck size={16} /> : <IconCopy size={16} />}
                  </ActionIcon>
                </Tooltip>
              )}
            </CopyButton>
          </Group>
        )}

        <TextInput
          label="Enter TOTP Code"
          value={totpCode}
          onChange={(e) => setTotpCode(e.target.value)}
          placeholder="000000"
          required
          mb="md"
        />

         <Group justify="center">
           <Button onClick={() => { void handleVerify(); }} loading={verifying}>
             Verify and Complete Setup
           </Button>
         </Group>
      </Paper>
    </Container>
  );
}

export default MFAOnboardingPage;
