import { useState, useEffect, useRef } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { useForm } from "@mantine/form";
import {
  TextInput,
  PasswordInput,
  Button,
  Title,
  Group,
  MultiSelect,
  Text,
  Container,
  Loader,
  Alert,
  Badge,
  SegmentedControl,
} from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { modals } from "@mantine/modals";
import { useAppUserStore } from "../stores/app-user-store";
import { useAppUserGroupStore } from "../stores/app-user-group-store";
import useAuthGuard from "../hooks/use-auth-guard";
import { useAuthStore } from "../stores/auth-store";
import { useProfileStore } from "../stores/profile-store";
import { confirmDelete } from "../utils/confirm-delete";
import { ChangePasswordModal } from "../components/change-password-modal";
import { InviteLinkModal } from "../components/invite-link-modal";
import { ROUTES } from "../constants/routes";
import {
  defaultUserFormData,
  type AccountStatus,
  type IssuedInvite,
} from "../types/user-type";
import { grants } from "../constants/grants";
import { PasswordGenerator } from "../components/password-generator";
import { MfaExemptField } from "../components/mfa-exempt-field";
import { reportError } from "../utils/report-error";


function UserEditPage() {
  const [searchParams] = useSearchParams();
  const {
    readUserById,
    createItem: createUser,
    updateItem: updateUser,
    deleteItem: deleteUser,
    changeOtherUserPassword,
    resetUserMFA,
    setUserBlocked,
    inviteUser,
    reissueInvite,
    revokeInvite,
  } = useAppUserStore();
  const {
    readList: readAppUserGroupList,
    itemList: appUserGroupList,
    loading: groupsLoading,
  } = useAppUserGroupStore();
  const navigate = useNavigate();

  const id = searchParams.get("appUserId") || "";
  const numericId = isNaN(Number(id)) ? null : Number(id);
  const isNew = id === "new";

  const form = useForm<typeof defaultUserFormData>({
    initialValues: defaultUserFormData,
    validate: {
      userName: (value) => (value.length < 3 ? 'Username must be at least 3 characters' : null),
      firstName: (value) => (value.length < 1 ? 'First name is required' : null),
      lastName: (value) => (value.length < 1 ? 'Last name is required' : null),
      email: (value) => (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value) ? 'Invalid email address' : null),
    },
  });
  const requiredGrant = isNew ? grants.CREATE_USER : grants.UPDATE_USER;
  const authState = useAuthGuard([requiredGrant, grants.READ_USER]);
  const grantList = useAuthStore((state) => state.grantList);
  const ownUserName = useAuthStore((state) => state.userName);
  // Whether the server lets the API set mfaExempt (B097); the profile is
  // loaded app-wide (app.tsx), and until it is the checkbox stays hidden.
  const allowMfaExemptViaApi = useProfileStore(
    (state) => state.profile?.allowMfaExemptViaApi === true,
  );
  // The route id the form was last loaded for, and what that load found;
  // a new ?appUserId= on the same page reloads the form.
  const requestedId = useRef<string | null>(null);
  const [loadResult, setLoadResult] = useState<{
    id: string;
    found: boolean;
  } | null>(null);
  const [passwordModalOpen, setPasswordModalOpen] = useState(false);
  const [passwordChangeLoading, setPasswordChangeLoading] = useState(false);
  const [passwordChangeError, setPasswordChangeError] = useState<
    string | undefined
  >();
  const [resetMFALoading, setResetMFALoading] = useState(false);
  const [accountStatus, setAccountStatus] =
    useState<AccountStatus>("active");
  const [blockLoading, setBlockLoading] = useState(false);
  // The loaded login's mfaExempt as stored, not as edited.
  const [storedMfaExempt, setStoredMfaExempt] = useState(false);
  // A new user is onboarded by an invite link (they choose their own
  // password) or with a password the admin sets (B035).
  const [onboarding, setOnboarding] = useState<"invite" | "password">(
    "invite",
  );
  const [issuedInvite, setIssuedInvite] = useState<IssuedInvite | null>(
    null,
  );
  const [inviteActionLoading, setInviteActionLoading] = useState(false);

  useEffect(() => {
    if (authState !== "YES" || requestedId.current === id) return;
    requestedId.current = id;
    void readAppUserGroupList().catch((err) => reportError(err, "user edit"));
    if (isNew || !numericId) {
      form.reset();
      return;
    }
    void readUserById(numericId)
      .then((user) => {
        if (requestedId.current !== id) return; // a later id took over
        if (user) {
          form.setValues({
            userName: user.userName,
            firstName: user.firstName,
            lastName: user.lastName,
            email: user.email,
            appUserGroupMembershipList: user.appUserGroupMembershipList,
            mfaExempt: user.mfaExempt,
            password: "",
            updateTs: user.updateTs.toString(),
          });
          setAccountStatus(user.accountStatus);
          setStoredMfaExempt(user.mfaExempt);
        }
        setLoadResult({ id, found: user !== null });
      })
      .catch((error: unknown) => {
        if (requestedId.current !== id) return;
        reportError(error, "loading user");
        notifications.show({
          title: "Error",
          message: "Failed to load user",
          color: "red",
          autoClose: 5000,
          withCloseButton: true,
        });
        setLoadResult({ id, found: false });
      });
    // `form` is a new object every render; requestedId makes this run once
    // per route id.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [authState, id, isNew, numericId, readAppUserGroupList, readUserById]);

  //============================^ HOOK LINE ^====================

  if (authState === "CHECKING") {
    return <Container size={1280} px={32} py={32} mx="auto" ta="center"><Loader /></Container>;
  }
  if (authState !== "YES") {
    return <Container size={1280} px={32} py={32} mx="auto"><Alert color="red">Authentication required.</Alert></Container>;
  }

  if (!isNew && numericId && loadResult?.id !== id) {
    return <Container size={1280} px={32} py={32} mx="auto" ta="center"><Loader /></Container>;
  }
  // Never render the form for a record that was not loaded.
  if (!isNew && (!numericId || !loadResult?.found)) {
    return (
      <Container size={1280} px={32} py={32} mx="auto">
        <Title order={1} mb="lg">Edit User</Title>
        <Alert color="red">This user could not be loaded.</Alert>
      </Container>
    );
  }

  const handleSubmit = async (values: typeof defaultUserFormData) => {
    const data = {
      userName: values.userName,
      firstName: values.firstName,
      lastName: values.lastName,
      email: values.email,
      appUserGroupMembershipList: values.appUserGroupMembershipList,
      mfaExempt: values.mfaExempt,
      password: values.password || undefined,
      updateTs: !isNew ? Number(values.updateTs) : undefined,
    };

    try {
      if (isNew && onboarding === "invite") {
        // The link is shown once, in the modal; closing it goes back to
        // the list.
        setIssuedInvite(
          await inviteUser({
            userName: data.userName,
            firstName: data.firstName,
            lastName: data.lastName,
            email: data.email,
            appUserGroupMembershipList: data.appUserGroupMembershipList,
            mfaExempt: data.mfaExempt,
          }),
        );
        return;
      }
      if (isNew) {
        await createUser(data);
        notifications.show({
          title: "Success",
          message: "User created successfully",
          color: "green",
          autoClose: 3000,
          withCloseButton: false,
        });
      } else {
        await updateUser({ appUserId: Number(id), ...data });
        notifications.show({
          title: "Success",
          message: "User updated successfully",
          color: "green",
          autoClose: 3000,
          withCloseButton: false,
        });
      }
      void navigate(ROUTES.APP_USER_LIST);
    } catch (error) {
      reportError(error, "saving user");
      notifications.show({
        title: "Error",
        message: "Failed to save user",
        color: "red",
        autoClose: 5000,
        withCloseButton: true,
      });
    }
  };

  const handleDeleteClick = () => {
    confirmDelete("user", () => void handleDeleteConfirm());
  };

  const handleDeleteConfirm = async () => {
    if (numericId) {
      try {
        await deleteUser(numericId, Number(form.values.updateTs));
        notifications.show({
          title: "Success",
          message: "User deleted successfully",
          color: "green",
          autoClose: 3000,
          withCloseButton: false,
        });
        void navigate(ROUTES.APP_USER_LIST);
      } catch (error) {
        reportError(error, "deleting user");
        notifications.show({
          title: "Error",
          message: "Failed to delete user",
          color: "red",
          autoClose: 5000,
          withCloseButton: true,
        });
      }
    }
  };

  const handlePasswordChange = async (
    _oldPassword: string,
    newPassword: string,
  ) => {
    if (!numericId) return;

    setPasswordChangeLoading(true);
    setPasswordChangeError(undefined);
    try {
      await changeOtherUserPassword(numericId, newPassword);
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

  const handleResetMFA = async () => {
    if (!numericId) return;

    setResetMFALoading(true);
    setPasswordChangeError(undefined);
    try {
      await resetUserMFA(numericId, parseFloat(form.values.updateTs) || 0);
      notifications.show({
        title: "Success",
        message: "MFA reset successfully. User must re-setup MFA on next login.",
        color: "green",
      });
      // Refresh user data
      if (numericId) {
        await readUserById(numericId);
      }
    } catch (err) {
      notifications.show({
        title: "Error",
        message: err instanceof Error ? err.message : "Failed to reset MFA",
        color: "red",
      });
    } finally {
      setResetMFALoading(false);
    }
  };

  // The block changes the row's update_ts: re-read it so a later Save does
  // not answer 409.
  const handleSetBlocked = async (blocked: boolean) => {
    if (!numericId) return;

    setBlockLoading(true);
    try {
      await setUserBlocked(form.values.userName, blocked);
      notifications.show({
        title: blocked ? "User blocked" : "User unblocked",
        message: blocked
          ? "They cannot sign in, and their open sessions are ended."
          : "They can sign in again.",
        color: blocked ? "orange" : "green",
      });
      const user = await readUserById(numericId);
      if (user) {
        form.setFieldValue("updateTs", user.updateTs.toString());
        setAccountStatus(user.accountStatus);
      }
    } catch (err) {
      notifications.show({
        title: "Error",
        message:
          err instanceof Error ? err.message : "Failed to change the block",
        color: "red",
      });
    } finally {
      setBlockLoading(false);
    }
  };

  const handleReissueInvite = async () => {
    if (!numericId) return;
    setInviteActionLoading(true);
    try {
      setIssuedInvite(await reissueInvite(numericId, form.values.userName));
    } catch (err) {
      notifications.show({
        title: "Error",
        message:
          err instanceof Error
            ? err.message
            : "Failed to issue a new invite link",
        color: "red",
      });
    } finally {
      setInviteActionLoading(false);
    }
  };

  const handleRevokeInvite = async () => {
    if (!numericId) return;
    setInviteActionLoading(true);
    try {
      await revokeInvite(numericId);
      notifications.show({
        title: "Invite link revoked",
        message:
          "The link no longer works. Issue a new one to invite this user " +
          "again.",
        color: "green",
      });
    } catch (err) {
      notifications.show({
        title: "Error",
        message:
          err instanceof Error
            ? err.message
            : "Failed to revoke the invite link",
        color: "red",
      });
    } finally {
      setInviteActionLoading(false);
    }
  };

  const isBlocked = !isNew && accountStatus === "blocked";
  const isInvited = !isNew && accountStatus === "invited";

  const groupOptions = appUserGroupList.map((g) => ({
    value: g.appUserGroupId.toString(),
    label: g.groupName,
  }));

  return (
    <Container size={1280} px={32} py={32} mx="auto">
      <Title order={1} mb="lg">
        {isNew ? "Add User" : "Edit User"}
        {isBlocked && (
          <Badge color="red" variant="light" ml="md">
            Blocked
          </Badge>
        )}
        {isInvited && (
          <Badge color="blue" variant="light" ml="md">
            Invited
          </Badge>
        )}
      </Title>
      <form onSubmit={form.onSubmit(handleSubmit)}>
        <TextInput
          label="Username"
          {...form.getInputProps('userName')}
          mb="md"
        />
        <TextInput
          label="First Name"
          {...form.getInputProps('firstName')}
          mb="md"
        />
        <TextInput
          label="Last Name"
          {...form.getInputProps('lastName')}
          mb="md"
        />
        <TextInput
          label="Email"
          {...form.getInputProps('email')}
          mb="md"
        />
        {isNew && (
          <SegmentedControl
            value={onboarding}
            onChange={(value) =>
              setOnboarding(value as "invite" | "password")
            }
            data={[
              { value: "invite", label: "Send an invite link" },
              { value: "password", label: "Set a password" },
            ]}
            mb="xs"
          />
        )}
        {isNew && onboarding === "invite" && (
          <Text size="sm" c="dimmed" mb="md">
            You get a one-time link to send them. They choose their own
            password when they open it.
          </Text>
        )}
        {isNew && onboarding === "password" && (
          <>
            <PasswordInput
              label="Password"
              {...form.getInputProps('password')}
              mb="md"
            />
            <PasswordGenerator
              onPasswordGenerated={(pwd) =>
                form.setFieldValue('password', pwd)
              }
            />
          </>
        )}
        <MultiSelect
          label="Groups"
          data={groupOptions}
          value={form.values.appUserGroupMembershipList.map(String)}
          onChange={(values) => form.setFieldValue('appUserGroupMembershipList', values.map(Number))}
          disabled={groupsLoading}
          mb="md"
        />
        <MfaExemptField
          allowSetting={allowMfaExemptViaApi}
          storedExempt={!isNew && storedMfaExempt}
          checked={form.values.mfaExempt}
          onChange={(checked) => form.setFieldValue('mfaExempt', checked)}
        />
        <Group>
          <Button type="submit">
            {isNew && onboarding === "invite"
              ? "Create and get invite link"
              : "Save"}
          </Button>
           {isInvited && grantList.includes(grants.UPDATE_USER) && (
             <>
               <Button
                 variant="outline"
                 loading={inviteActionLoading}
                 onClick={() => void handleReissueInvite()}
               >
                 New invite link
               </Button>
               <Button
                 variant="outline"
                 color="orange"
                 loading={inviteActionLoading}
                 onClick={() => void handleRevokeInvite()}
               >
                 Revoke link
               </Button>
             </>
           )}
           {!isNew && !isInvited && (
             <Button
               variant="outline"
               onClick={() => setPasswordModalOpen(true)}
             >
               Change Password
             </Button>
           )}
           {!isNew && !isInvited &&
             grantList.includes(grants.RESET_MFA_TOKEN) && (
             <Button
               color="orange"
               loading={resetMFALoading}
               onClick={() =>
                 modals.openConfirmModal({
                   title: "Reset MFA",
                   children: (
                     <Text size="sm">
                       This will clear the user's MFA setup, requiring them to re-configure it on their next login. This action cannot be undone. Are you sure?
                     </Text>
                   ),
                   labels: { confirm: "Reset MFA", cancel: "Cancel" },
                   confirmProps: { color: "orange" },
                   onConfirm: () => void handleResetMFA(),
                 })
               }
             >
               Reset MFA
             </Button>
           )}
           {!isNew && !isInvited && grantList.includes(grants.BLOCK_USER) &&
             form.values.userName !== ownUserName && (
             <Button
               color={isBlocked ? "green" : "orange"}
               variant="outline"
               loading={blockLoading}
               onClick={() =>
                 modals.openConfirmModal({
                   title: isBlocked ? "Unblock user" : "Block user",
                   children: (
                     <Text size="sm">
                       {isBlocked
                         ? "They will be able to sign in again."
                         : "They will not be able to sign in, and their open sessions end at once. Unblock to undo."}
                     </Text>
                   ),
                   labels: {
                     confirm: isBlocked ? "Unblock" : "Block",
                     cancel: "Cancel",
                   },
                   confirmProps: { color: isBlocked ? "green" : "orange" },
                   onConfirm: () => void handleSetBlocked(!isBlocked),
                 })
               }
             >
               {isBlocked ? "Unblock" : "Block"}
             </Button>
           )}
           {!isNew && form.values.userName !== ownUserName && (
             <Button color="red" onClick={handleDeleteClick}>
               Delete
             </Button>
           )}
          <Button
            variant="outline"
            onClick={() => void navigate(ROUTES.APP_USER_LIST)}
          >
            Cancel
          </Button>
        </Group>
      </form>
      <ChangePasswordModal
        opened={passwordModalOpen}
        onSubmit={(oldPassword, newPassword) => {
          void handlePasswordChange(oldPassword, newPassword);
        }}
        onClose={() => {
          setPasswordModalOpen(false);
          setPasswordChangeError(undefined);
        }}
        mode="other"
        loading={passwordChangeLoading}
        error={passwordChangeError}
      />
      <InviteLinkModal
        invite={issuedInvite}
        onClose={() => {
          setIssuedInvite(null);
          if (isNew) void navigate(ROUTES.APP_USER_LIST);
        }}
      />
    </Container>
  );
}

export default UserEditPage;
