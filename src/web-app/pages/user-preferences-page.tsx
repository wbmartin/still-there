import { useState, useEffect, useRef } from "react";
import {
  Button,
  Title,
  Group,
  Text,
  Select,
  Alert,
  Container,
  Stack,
  Loader,
  TextInput,
  NumberInput,
} from "@mantine/core";
import { useForm } from "@mantine/form";
import { notifications } from "@mantine/notifications";
import { useNavigate } from "react-router-dom";
import { useAuthStore } from "../stores/auth-store";
import { clearAllCaches } from "../utils/cache-utils";
import { ROUTES } from "../constants/routes";
import useAuthGuard from "../hooks/use-auth-guard";
import { grants } from "../constants/grants";
import { ChangePasswordModal } from "../components/change-password-modal";
import { useProfileStore } from "../stores/profile-store";
import {
  resolvePreference,
  type ColorScheme,
  type Profile,
} from "../types/profile-type";
import { reportError } from "../utils/report-error";

const colorSchemeOptions: { value: ColorScheme; label: string }[] = [
  { value: "auto", label: "Follow the system" },
  { value: "light", label: "Light" },
  { value: "dark", label: "Dark" },
];

const isColorScheme = (value: string): value is ColorScheme =>
  colorSchemeOptions.some((option) => option.value === value);

const profileFormValues = (profile: Profile) => {
  const preference = resolvePreference(profile.preferenceJsonb);
  return {
    firstName: profile.firstName,
    lastName: profile.lastName,
    email: profile.email,
    colorScheme: preference.colorScheme as string,
    rowsPerPage: preference.rowsPerPage,
  };
};

function UserPreferencesPage() {
  const authState = useAuthGuard([grants.LOGON]);
  const navigate = useNavigate();
  const {
    profile,
    readProfile,
    updateProfile,
    loading: profileLoading,
  } = useProfileStore();
  const profileForm = useForm({
    initialValues: {
      firstName: "",
      lastName: "",
      email: "",
      colorScheme: "auto",
      rowsPerPage: 15,
    },
    validate: {
      firstName: (value) => (value.trim() ? null : "First name is required"),
      lastName: (value) => (value.trim() ? null : "Last name is required"),
      email: (value) =>
        /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value) ? null : "Invalid email address",
      rowsPerPage: (value) =>
        Number.isInteger(value) && value >= 5 && value <= 200
          ? null
          : "Between 5 and 200",
    },
  });
  const hasFetched = useRef(false);
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

  useEffect(() => {
    if (!hasFetched.current && authState === "YES") {
      hasFetched.current = true;
      // app.tsx loads the profile for the session; re-read here so the
      // form always starts from the row as stored.
      void readProfile()
        .then((loaded) => profileForm.setValues(profileFormValues(loaded)))
        .catch((err) => reportError(err, "profile"));
    }
    // `profileForm` is a new object every render; hasFetched makes this a
    // one-shot.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [authState, readProfile]);

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

  const handleProfileSave = async (values: typeof profileForm.values) => {
    if (!profile) return;
    try {
      const saved = await updateProfile({
        firstName: values.firstName.trim(),
        lastName: values.lastName.trim(),
        email: values.email.trim(),
        preferenceJsonb: {
          colorScheme: isColorScheme(values.colorScheme)
            ? values.colorScheme
            : "auto",
          rowsPerPage: values.rowsPerPage,
        },
        updateTs: profile.updateTs,
      });
      profileForm.setValues(profileFormValues(saved));
      profileForm.resetDirty(profileFormValues(saved));
      notifications.show({
        title: "Saved",
        message: "Profile and preferences saved",
        color: "green",
      });
    } catch (err) {
      reportError(err, "saving profile");
      notifications.show({
        title: "Error",
        message: err instanceof Error ? err.message : "Failed to save profile",
        color: "red",
      });
    }
  };

  const contextOptions = availableSecurityContexts.map((ctx) => ({
    value: ctx.toString(),
    label: `Context ${ctx}`,
  }));

  return (
    <Container size={1280} px={32} py={32} mx="auto" maw="600px">
      <Title order={1} mb="lg">
        Profile
      </Title>

      <form onSubmit={profileForm.onSubmit((values) => void handleProfileSave(values))}>
        <Stack>
          <Group mb="xs">
            <Text fw={500}>Username:</Text>
            <Text>{userName}</Text>
            <Text fw={500} ml="lg">Security Context:</Text>
            <Text>{securityContext}</Text>
          </Group>
          <Group grow>
            <TextInput label="First name" {...profileForm.getInputProps("firstName")} />
            <TextInput label="Last name" {...profileForm.getInputProps("lastName")} />
          </Group>
          <TextInput label="Email" {...profileForm.getInputProps("email")} />
        </Stack>

        <Title order={3} mt="2rem" mb="sm">
          Preferences
        </Title>
        <Group grow align="flex-start">
          <Select
            label="Colour scheme"
            data={colorSchemeOptions}
            allowDeselect={false}
            {...profileForm.getInputProps("colorScheme")}
          />
          <NumberInput
            label="Rows per page"
            description="List pages"
            min={5}
            max={200}
            {...profileForm.getInputProps("rowsPerPage")}
          />
        </Group>
        <Group mt="md">
          <Button type="submit" loading={profileLoading} disabled={!profile}>
            Save
          </Button>
        </Group>
      </form>

      <Title order={3} mt="2rem" mb="sm">
        Security
      </Title>
      {availableSecurityContexts.length > 1 && (
        <Stack>
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

      <Stack mt="md">
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
          Session
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

      <Stack mt="md">
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
