import { useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useForm } from "@mantine/form";
import {
  Alert,
  Anchor,
  Button,
  Center,
  Container,
  Loader,
  Paper,
  PasswordInput,
  Text,
  Title,
} from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { apiJson, jsonBody } from "../utils/api";
import { formatInviteExpiry, readInviteToken } from "../utils/invite-link";
import { API_ENDPOINTS } from "../constants/api";
import { ROUTES } from "../constants/routes";
import { APP_NAME } from "../constants/app";
import { useAuthStore } from "../stores/auth-store";
import type { ApiResponse } from "../types/api";
import type { ApiInvitePreview } from "../types/user-type";

// The invite link's landing page (B035): `/invite#<token>`, opened by
// someone who has never signed in. It reads who the link is for, takes a
// password, redeems the link, and then signs in with that password, so MFA
// enrolment is the ordinary first-login path (mfa-onboarding-page), not a
// second copy of it here.
//
// Both calls are `anonymous: true`: there is no session, and a refusal
// means the link is dead, never that a session expired.

// The API's floor (handler minPasswordLength).
const MIN_PASSWORD_LENGTH = 8;

type PageState =
  | { kind: "checking"; token: string }
  | { kind: "ready"; token: string; preview: ApiInvitePreview }
  | { kind: "dead"; message: string };

function InvitePage() {
  const navigate = useNavigate();
  const { authenticateWithUsernamePassword } = useAuthStore();
  // The token is read once, from the fragment; a missing or mangled one is
  // dead before any call.
  const [pageState, setPageState] = useState<PageState>(() => {
    const token = readInviteToken(window.location.hash);
    return token
      ? { kind: "checking", token }
      : {
          kind: "dead",
          message:
            "This invite link is incomplete. Open it straight from the " +
            "message you were sent, or ask your administrator for a new one.",
        };
  });
  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);
  // A ref, not state: the check runs once per page load.
  const hasChecked = useRef(false);

  const form = useForm({
    initialValues: { password: "", confirmPassword: "" },
    validate: {
      password: (value) =>
        value.length < MIN_PASSWORD_LENGTH
          ? `Password must be at least ${MIN_PASSWORD_LENGTH} characters`
          : null,
      confirmPassword: (value, values) =>
        value !== values.password ? "Passwords do not match" : null,
    },
  });

  const checkingToken =
    pageState.kind === "checking" ? pageState.token : null;
  useEffect(() => {
    if (!checkingToken || hasChecked.current) return;
    hasChecked.current = true;
    const token = checkingToken;
    void (async () => {
      try {
        const answer = await apiJson<ApiResponse<ApiInvitePreview>>(
          API_ENDPOINTS.INVITE_CHECK,
          { method: "POST", ...jsonBody({ token }) },
          { anonymous: true },
        );
        setPageState({ kind: "ready", token, preview: answer.data });
      } catch (err) {
        setPageState({
          kind: "dead",
          message:
            err instanceof Error
              ? err.message
              : "This invite link cannot be used.",
        });
      }
    })();
  }, [checkingToken]);

  //============================^ HOOK LINE ^====================

  const handleSubmit = form.onSubmit(async (values) => {
    if (pageState.kind !== "ready") return;
    setSubmitting(true);
    setSubmitError(null);
    let userName: string;
    try {
      const answer = await apiJson<ApiResponse<{ userName: string }>>(
        API_ENDPOINTS.INVITE_REDEEM,
        {
          method: "POST",
          ...jsonBody({ token: pageState.token, password: values.password }),
        },
        { anonymous: true },
      );
      userName = answer.data.userName;
    } catch (err) {
      setSubmitError(
        err instanceof Error ? err.message : "The invite could not be accepted.",
      );
      setSubmitting(false);
      return;
    }

    // The link is spent: take the token out of the address bar and the
    // history entry.
    window.history.replaceState(null, "", ROUTES.INVITE);

    await authenticateWithUsernamePassword(userName, values.password);
    const state = useAuthStore.getState();
    if (state.authState === "MFA") {
      void navigate(state.mfaEnrolled ? ROUTES.MFA_VERIFY : ROUTES.MFA_SETUP);
    } else if (state.authState === "YES") {
      void navigate(ROUTES.DASHBOARD);
    } else {
      notifications.show({
        title: "Password set",
        message: `Sign in as ${userName} with your new password.`,
        color: "green",
      });
      void navigate(ROUTES.LOGIN);
    }
  });

  return (
    <Center style={{ minHeight: "100vh" }}>
      <Container size="xs" w="100%">
        <Paper shadow="md" p="xl">
          <Title order={2} mb="md" ta="center">
            Join {APP_NAME}
          </Title>
          {pageState.kind === "checking" && (
            <Center>
              <Loader />
            </Center>
          )}
          {pageState.kind === "dead" && (
            <>
              <Alert color="red" variant="light" mb="md">
                {pageState.message}
              </Alert>
              <Text ta="center" size="sm">
                Already set a password?{" "}
                <Anchor onClick={() => void navigate(ROUTES.LOGIN)}>
                  Sign in
                </Anchor>
              </Text>
            </>
          )}
          {pageState.kind === "ready" && (
            <form onSubmit={handleSubmit}>
              <Text mb="xs">Welcome, {pageState.preview.firstName}.</Text>
              <Text size="sm" c="dimmed" mb="md">
                Choose a password for the user name{" "}
                <b>{pageState.preview.userName}</b>. This link works once
                and expires {formatInviteExpiry(pageState.preview.expiresTs)}.
              </Text>
              <PasswordInput
                label="Password"
                autoComplete="new-password"
                {...form.getInputProps("password")}
                mb="md"
              />
              <PasswordInput
                label="Confirm password"
                autoComplete="new-password"
                {...form.getInputProps("confirmPassword")}
                mb="md"
              />
              {submitError && (
                <Alert color="red" variant="light" mb="md">
                  {submitError}
                </Alert>
              )}
              <Button type="submit" fullWidth loading={submitting}>
                Set password and continue
              </Button>
            </form>
          )}
        </Paper>
      </Container>
    </Center>
  );
}

export default InvitePage;
