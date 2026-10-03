import { useState, useEffect, useRef } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { useForm } from "@mantine/form";
import { ROUTES } from "../constants/routes";
import { grants as grantConstants } from "../constants/grants";
import {
  TextInput,
  Button,
  Title,
  Group,
  MultiSelect,
  List,
  Text,
  Container,
  Stack,
  Loader,
  Alert,
} from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { useAppUserGroupStore } from "../stores/app-user-group-store";
import { useAppUserStore } from "../stores/app-user-store";
import useAuthGuard from "../hooks/use-auth-guard";
import { confirmDelete } from "../utils/confirm-delete";
import { AddRemoveUsersFromGroupModal } from "../components/add-remove-users-from-group-modal";
import { defaultAppUserGroupFormData } from "../types/app-user-group-type";
import { reportError } from "../utils/report-error";

function AppUserGroupEditPage() {
  const [searchParams] = useSearchParams();
  const appUserGroupId = searchParams.get("appUserGroupId") || "";
  const isNew = appUserGroupId === "new";
  const requiredGrant = isNew
    ? grantConstants.CREATE_USER_GROUP
    : grantConstants.UPDATE_USER_GROUP;
  const authState = useAuthGuard([requiredGrant]);
  const {
    readAppUserGroupById,
    createItem: createAppUserGroup,
    updateItem: updateAppUserGroup,
    deleteItem: deleteAppUserGroup,
  } = useAppUserGroupStore();
  const {
    itemList: userList,
    readList: readUserList,
    addUserListToGroup,
    removeUserListFromGroup,
  } = useAppUserStore();
  const navigate = useNavigate();

  const numericId = isNaN(Number(appUserGroupId))
    ? null
    : Number(appUserGroupId);

  const form = useForm<typeof defaultAppUserGroupFormData>({
    initialValues: defaultAppUserGroupFormData,
    validate: {
      groupName: (value) => (value.length < 1 ? 'Group name is required' : null),
    },
  });

  const grants = grantConstants.getGrantList();

  // The route id the form was last loaded for, and what that load found;
  // a new ?appUserGroupId= on the same page reloads the form.
  const requestedId = useRef<string | null>(null);
  const [loadResult, setLoadResult] = useState<{
    id: string;
    found: boolean;
  } | null>(null);
  const [userModalOpen, setUserModalOpen] = useState(false);
  const [userModalMode, setUserModalMode] = useState<"add" | "remove">("add");

  useEffect(() => {
    if (authState !== "YES" || requestedId.current === appUserGroupId) {
      return;
    }
    requestedId.current = appUserGroupId;
    if (isNew || !numericId) {
      form.reset();
      return;
    }
    // Users are only needed for the add/remove-member modal
    void readUserList().catch((err) => reportError(err, "user group edit"));
    void readAppUserGroupById(numericId)
      .then((existingGroup) => {
        // A later id took over.
        if (requestedId.current !== appUserGroupId) return;
        if (existingGroup) {
          form.setValues({
            groupName: existingGroup.groupName,
            userGrantList: existingGroup.userGrantList,
            updateTs: existingGroup.updateTs.toString(),
          });
        }
        setLoadResult({ id: appUserGroupId, found: existingGroup !== null });
      })
      .catch((error: unknown) => {
        if (requestedId.current !== appUserGroupId) return;
        reportError(error, "loading user group");
        notifications.show({
          title: "Error",
          message: "Failed to load user group",
          color: "red",
          autoClose: 5000,
          withCloseButton: true,
        });
        setLoadResult({ id: appUserGroupId, found: false });
      });
    // `form` is a new object every render; requestedId makes this run once
    // per route id.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [authState, appUserGroupId, isNew, numericId, readUserList,
    readAppUserGroupById]);

  //============================^ HOOK LINE ^====================

  if (authState === "CHECKING") {
    return <Container size={1280} px={32} py={32} mx="auto" ta="center"><Loader /></Container>;
  }
  if (authState !== "YES") {
    return <Container size={1280} px={32} py={32} mx="auto"><Alert color="red">Authentication required.</Alert></Container>;
  }

  if (!isNew && numericId && loadResult?.id !== appUserGroupId) {
    return <Container size={1280} px={32} py={32} mx="auto" ta="center"><Loader /></Container>;
  }
  // Never render the form for a record that was not loaded.
  if (!isNew && (!numericId || !loadResult?.found)) {
    return (
      <Container size={1280} px={32} py={32} mx="auto">
        <Title order={1} mb="lg">Edit User Group</Title>
        <Alert color="red">This user group could not be loaded.</Alert>
      </Container>
    );
  }

  const handleSubmit = async (values: typeof defaultAppUserGroupFormData) => {
    const data = {
      groupName: values.groupName,
      userGrantList: values.userGrantList,
      updateTs: Number(values.updateTs),
    };

    try {
      if (isNew) {
        await createAppUserGroup({
          groupName: values.groupName,
          userGrantList: values.userGrantList,
        });
        notifications.show({
          title: "Success",
          message: "User group created successfully",
          color: "green",
          autoClose: 3000,
          withCloseButton: false,
        });
      } else {
        await updateAppUserGroup({
          appUserGroupId: Number(appUserGroupId),
          ...data,
        });
        notifications.show({
          title: "Success",
          message: "User group updated successfully",
          color: "green",
          autoClose: 3000,
          withCloseButton: false,
        });
      }
      void navigate(ROUTES.APP_USER_GROUP_LIST);
    } catch (error) {
      reportError(error, "saving user group");
      notifications.show({
        title: "Error",
        message: "Failed to save user group",
        color: "red",
        autoClose: 5000,
        withCloseButton: true,
      });
    }
  };

  const handleDeleteClick = () => {
    confirmDelete("user group", () => void handleDeleteConfirm());
  };

  const handleDeleteConfirm = async () => {
    if (numericId && !isNew) {
      try {
        await deleteAppUserGroup(numericId, Number(form.values.updateTs));
        notifications.show({
          title: "Success",
          message: "User group deleted successfully",
          color: "green",
          autoClose: 3000,
          withCloseButton: false,
        });
        void navigate(ROUTES.APP_USER_GROUP_LIST);
      } catch (error) {
        reportError(error, "deleting user group");
        notifications.show({
          title: "Error",
          message: "Failed to delete user group",
          color: "red",
          autoClose: 5000,
          withCloseButton: true,
        });
      }
    }
  };

  const handleUserModalConfirm = async (userIdList: number[]) => {
    if (!numericId) return;
    const updateTsList = userIdList.map(
      (id) => userList.find((u) => u.appUserId === id)!.updateTs,
    );
    try {
      if (userModalMode === "add") {
        await addUserListToGroup(userIdList, numericId, updateTsList);
      } else {
        await removeUserListFromGroup(userIdList, numericId, updateTsList);
      }
      notifications.show({
        title: "Success",
        message: `Users ${userModalMode === "add" ? "added to" : "removed from"} group successfully`,
        color: "green",
        autoClose: 3000,
        withCloseButton: false,
      });
    } catch (error) {
      reportError(error, `${userModalMode}ing users`);
      notifications.show({
        title: "Error",
        message: `Failed to ${userModalMode} users`,
        color: "red",
        autoClose: 5000,
      });
    }
  };

  return (
    <Container size={1280} px={32} py={32} mx="auto">
      <Title order={1} mb="lg">
        Edit User Group
      </Title>
      <form onSubmit={form.onSubmit(handleSubmit)}>
        <TextInput
          label="Group Name"
          {...form.getInputProps('groupName')}
          mb="md"
        />
        <MultiSelect
          label="Roles"
          data={grants.map((g) => ({ value: g.id.toString(), label: g.text }))}
          value={form.values.userGrantList.map(String)}
          onChange={(values) => form.setFieldValue('userGrantList', values.map(Number))}
          mb="md"
        />
        <Group>
          <Button type="submit">Save</Button>
          <Button color="red" onClick={handleDeleteClick}>
            Delete
          </Button>
          <Button
            variant="outline"
            onClick={() => void navigate(ROUTES.APP_USER_GROUP_LIST)}
          >
            Cancel
          </Button>
        </Group>
      </form>

      {/* User Management Section */}
      <Stack mt="2rem">
        <Title order={2} mb="md">
          Group Members
        </Title>
        {(() => {
          const groupMemberList = userList.filter((u) =>
            u.appUserGroupMembershipList.includes(numericId || 0),
          );
          return groupMemberList.length > 0 ? (
            <List type="ordered" mb="md">
              {groupMemberList.map((user) => (
                <List.Item key={user.appUserId}>
                  {user.firstName} {user.lastName} ({user.userName})
                </List.Item>
              ))}
            </List>
          ) : (
            <Text c="dimmed" mb="md">
              No members
            </Text>
          );
        })()}
        <Group>
          <Button
            onClick={() => {
              setUserModalMode("add");
              setUserModalOpen(true);
            }}
          >
            Add Users
          </Button>
          <Button
            color="orange"
            onClick={() => {
              setUserModalMode("remove");
              setUserModalOpen(true);
            }}
          >
            Remove Users
          </Button>
        </Group>
      </Stack>

      <AddRemoveUsersFromGroupModal
        opened={userModalOpen}
        onClose={() => setUserModalOpen(false)}
        groupId={numericId}
        mode={userModalMode}
        onSubmit={handleUserModalConfirm}
      />
    </Container>
  );
}

export default AppUserGroupEditPage;
