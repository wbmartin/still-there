import { useState, useEffect, useRef } from "react";
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
import { apiJson, jsonBody } from "../utils/api";
import { API_ENDPOINTS } from "../constants/api";
import { useAuthStore } from "../stores/auth-store";
import { ROUTES } from "../constants/routes";
import type { LoginResponse } from "../types/auth-type";
import { reportError } from "../utils/report-error";

function MFAOnboardingPage() {
  const [secret, setSecret] = useState("");
  const [qrCodeImage, setQrCodeImage] = useState("");
  const [totpCode, setTotpCode] = useState("");
  const [loading, setLoading] = useState(false);
  const [verifying, setVerifying] = useState(false);
  const [mfaAlreadySetup, setMfaAlreadySetup] = useState(false);
  const navigate = useNavigate();
  const { tempToken } = useAuthStore();
  const hasFetched = useRef(false);

  useEffect(() => {
    if (!tempToken || hasFetched.current) return;
    hasFetched.current = true;

    const fetchSetup = async () => {
      setLoading(true);
      try {
        // anonymous: this runs on a temp token, before there is a session
        const data = await apiJson<{
          data: {
            secret: string;
            qrCodeImage: string;
            mfaAlreadySetup: boolean;
          };
        }>(
          API_ENDPOINTS.MFA_SETUP,
          { method: "POST", ...jsonBody({ tempToken }) },
          { anonymous: true },
        );
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
      // anonymous: a wrong code is a 401 that must not end the session
      const data = await apiJson<LoginResponse>(
        API_ENDPOINTS.MFA_VERIFY,
        { method: "POST", ...jsonBody({ tempToken, totpCode }) },
        { anonymous: true },
      );
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
      void navigate(ROUTES.DASHBOARD);
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
                onError={() => reportError(new Error("QR code image failed to load"), "mfa onboarding")}
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
