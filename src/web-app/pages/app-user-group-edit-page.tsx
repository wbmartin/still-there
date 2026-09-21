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
import type { AppUserGroup } from "../types/app-user-group-type";
import { DeleteConfirmModal } from "../components/delete-confirm-modal";
import AddRemoveUsersFromGroupModal from "../components/add-remove-users-from-group-modal";
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

  const hasFetched = useRef(false);
  const [deleteModalOpen, setDeleteModalOpen] = useState(false);
  const [userModalOpen, setUserModalOpen] = useState(false);
  const [userModalMode, setUserModalMode] = useState<"add" | "remove">("add");

  useEffect(() => {
    if (!hasFetched.current && numericId) {
      hasFetched.current = true;
      // Find group from store
      const groups = useAppUserGroupStore.getState().itemList;
      const existingGroup = groups.find(
        (g: AppUserGroup) => g.appUserGroupId === numericId,
      );
      if (existingGroup) {
        setTimeout(() => {
          form.setValues({
            groupName: existingGroup.groupName,
            userGrantList: existingGroup.userGrantList,
            updateTs: existingGroup.updateTs.toString(),
          });
        }, 0);
      }
      // Load users for the group management
      void readUserList().catch((err) => reportError(err, "user group edit"));
    }
  }, [numericId, readUserList]);

  //============================^ HOOK LINE ^====================

  if (authState === "CHECKING") {
    return <Container size={1280} px={32} py={32} mx="auto" ta="center"><Loader /></Container>;
  }
  if (authState !== "YES") {
    return <Container size={1280} px={32} py={32} mx="auto"><Alert color="red">Authentication required.</Alert></Container>;
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
      void navigate(`/${ROUTES.APP_USER_GROUP_LIST}`);
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
    setDeleteModalOpen(true);
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
        void navigate(`/${ROUTES.APP_USER_GROUP_LIST}`);
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
    setDeleteModalOpen(false);
  };

  const handleDeleteCancel = () => {
    setDeleteModalOpen(false);
  };

  const handleUserModalConfirm = async (userIds: number[]) => {
    if (!numericId) return;
    const updateTsList = userIds.map(
      (id) => userList.find((u) => u.appUserId === id)!.updateTs,
    );
    try {
      if (userModalMode === "add") {
        await addUserListToGroup(userIds, numericId, updateTsList);
      } else {
        await removeUserListFromGroup(userIds, numericId, updateTsList);
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
            onClick={() => void navigate(`/${ROUTES.APP_USER_GROUP_LIST}`)}
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

      <DeleteConfirmModal
        isOpen={deleteModalOpen}
        onConfirm={() => void handleDeleteConfirm()}
        onCancel={handleDeleteCancel}
        entityType="user group"
      />
      <AddRemoveUsersFromGroupModal
        isOpen={userModalOpen}
        onClose={() => setUserModalOpen(false)}
        groupId={numericId}
        mode={userModalMode}
        onConfirm={handleUserModalConfirm}
      />
    </Container>
  );
}

export default AppUserGroupEditPage;
