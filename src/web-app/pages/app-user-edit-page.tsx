import { useState, useEffect, useRef } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { useForm } from "@mantine/form";
import {
  TextInput,
  PasswordInput,
  Button,
  Title,
  Group,
  Checkbox,
  MultiSelect,
  Text,
  Container,
  Loader,
  Alert,
} from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { modals } from "@mantine/modals";
import { useAppUserStore } from "../stores/app-user-store";
import { useAppUserGroupStore } from "../stores/app-user-group-store";
import useAuthGuard from "../hooks/use-auth-guard";
import { useAuthStore } from "../stores/auth-store";
import { confirmDelete } from "../utils/confirm-delete";
import { ChangePasswordModal } from "../components/change-password-modal";
import { ROUTES } from "../constants/routes";
import { defaultUserFormData } from "../types/user-type";
import { grants } from "../constants/grants";
import { PasswordGenerator } from "../components/password-generator";
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
  const [userLoading, setUserLoading] = useState(!isNew && numericId !== null);
  const hasFetched = useRef(false);
  const [passwordModalOpen, setPasswordModalOpen] = useState(false);
  const [passwordChangeLoading, setPasswordChangeLoading] = useState(false);
  const [passwordChangeError, setPasswordChangeError] = useState<
    string | undefined
  >();
  const [resetMFALoading, setResetMFALoading] = useState(false);

  useEffect(() => {
    if (!hasFetched.current && authState === "YES") {
      hasFetched.current = true;
      void readAppUserGroupList().catch((err) => reportError(err, "user edit"));
      if (numericId && !isNew) {
        void readUserById(numericId)
          .then((user) => {
            if (user) {
              form.setValues({
                userName: user.userName,
                firstName: user.firstName,
                lastName: user.lastName,
                email: user.email,
                appUserGroupMembershipList: user.appUserGroupMembershipList,
                mfaExempt: user.mfaExempt,
                pwHash: "",
                updateTs: user.updateTs.toString(),
              });
            }
          })
          .catch((error: unknown) => {
            reportError(error, "loading user");
          })
          .finally(() => {
            setUserLoading(false);
          });
      }
    }
    // `form` is a new object every render; hasFetched makes this a one-shot.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [authState, numericId, isNew, readAppUserGroupList, readUserById]);

  //============================^ HOOK LINE ^====================

  if (authState === "CHECKING") {
    return <Container size={1280} px={32} py={32} mx="auto" ta="center"><Loader /></Container>;
  }
  if (authState !== "YES") {
    return <Container size={1280} px={32} py={32} mx="auto"><Alert color="red">Authentication required.</Alert></Container>;
  }

  if (userLoading) {
    return <Container size={1280} px={32} py={32} mx="auto" ta="center"><Loader /></Container>;
  }

  const handleSubmit = async (values: typeof defaultUserFormData) => {
    const data = {
      userName: values.userName,
      firstName: values.firstName,
      lastName: values.lastName,
      email: values.email,
      appUserGroupMembershipList: values.appUserGroupMembershipList,
      mfaExempt: values.mfaExempt,
      pwHash: values.pwHash || undefined,
      updateTs: !isNew ? Number(values.updateTs) : undefined,
    };

    try {
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

  const groupOptions = appUserGroupList.map((g) => ({
    value: g.appUserGroupId.toString(),
    label: g.groupName,
  }));

  return (
    <Container size={1280} px={32} py={32} mx="auto">
      <Title order={1} mb="lg">
        {isNew ? "Add User" : "Edit User"}
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
          <>
            <PasswordInput
              label="Password"
              {...form.getInputProps('pwHash')}
              mb="md"
            />
            <PasswordGenerator
              onPasswordGenerated={(pwd) =>
                form.setFieldValue('pwHash', pwd)
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
        <Checkbox
          label="MFA Exempt"
          {...form.getInputProps('mfaExempt')}
          mb="md"
        />
        <Group>
          <Button type="submit">Save</Button>
           {!isNew && (
             <Button
               variant="outline"
               onClick={() => setPasswordModalOpen(true)}
             >
               Change Password
             </Button>
           )}
           {!isNew && grantList.includes(grants.RESET_MFA_TOKEN) && (
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
           {!isNew && (
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
    </Container>
  );
}

export default UserEditPage;
