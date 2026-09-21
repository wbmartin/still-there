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
} from "@mantine/core";
import { IconCopy, IconCheck } from "@tabler/icons-react";
import { notifications } from "@mantine/notifications";
import { apiRequest } from "../utils/api";
import { API_ENDPOINTS } from "../constants/api";
import { useAuthStore } from "../stores/auth-store";
import { ROUTES } from "../constants/routes";
import type { LoginResponse } from "../types/auth-type";

function MFAOnboardingPage() {
  const [secret, setSecret] = useState("");
  const [qrCodeImage, setQrCodeImage] = useState("");
  const [totpCode, setTotpCode] = useState("");
  const [loading, setLoading] = useState(false);
  const [verifying, setVerifying] = useState(false);
  const [mfaAlreadySetup, setMfaAlreadySetup] = useState(false);
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
          data: {
            secret: string;
            qrCodeImage: string;
            mfaAlreadySetup: boolean;
          };
        };
        setSecret(data.data.secret);
        setQrCodeImage(data.data.qrCodeImage);
        setMfaAlreadySetup(data.data.mfaAlreadySetup || false);
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
      if (!mfaAlreadySetup) {
        notifications.show({
          title: "Success",
          message: "MFA setup completed successfully",
          color: "green",
        });
      }
      void navigate(`/${ROUTES.DASHBOARD}`);
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
          {mfaAlreadySetup ? "Verify Multi-Factor Authentication" : "Set up Multi-Factor Authentication"}
        </Title>
        <Text mb="md">
          {mfaAlreadySetup
            ? "Enter the verification code from your authenticator app."
            : "Scan the QR code with your authenticator app, or manually enter the secret key."
          }
        </Text>

        {!mfaAlreadySetup && qrCodeImage && (
          <Group justify="center" mb="md">
            <div style={{ border: '1px solid #ccc', borderRadius: '8px', padding: '10px', display: 'inline-block' }}>
              <img
                src={qrCodeImage}
                alt="QR Code for MFA Setup - Scan with your authenticator app"
                style={{ width: '200px', height: '200px', objectFit: 'contain' }}
                onError={() => console.error('QR Code image failed to load')}
              />
            </div>
          </Group>
        )}

        {!mfaAlreadySetup && secret && (
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
          <Button
            onClick={() => {
              void handleVerify();
            }}
            loading={verifying}
          >
            Verify Code
          </Button>
        </Group>
      </Paper>
    </Container>
  );
}

export default MFAOnboardingPage;
