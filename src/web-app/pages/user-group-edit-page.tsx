import { useState, useEffect, useRef } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { ROUTES } from "../constants/routes";
import {
  TextInput,
  Button,
  Title,
  Group,
  MultiSelect,
  List,
  Text,
} from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { useUserGroupsStore } from "../stores/user-groups-store";
import { useAppUserStore } from "../stores/app-user-store";
import useAuthGuard from "../hooks/use-auth-guard";
import { DeleteConfirmModal } from "../components/DeleteConfirmModal";
import AddRemoveUsersFromGroupModal from "../components/AddRemoveUsersFromGroupModal";
import type { UserGroupFormData } from "../types/user-group-type";
import { defaultUserGroupFormData } from "../types/user-group-type";

function UserGroupEditPage() {
  const authState = useAuthGuard();
  const [searchParams] = useSearchParams();
  const { createUserGroup, updateUserGroup, deleteUserGroup, readGrantList } =
    useUserGroupsStore();
  const {
    userList,
    retrieveUserList,
    addUserListToGroup,
    removeUserListFromGroup,
  } = useAppUserStore();
  const navigate = useNavigate();

  const userGroupId = searchParams.get("userGroupId") || "";
  const isNew = userGroupId === "new";
  const numericId = isNaN(Number(userGroupId)) ? null : Number(userGroupId);

  const [formData, setFormData] = useState<UserGroupFormData>(
    defaultUserGroupFormData,
  );

  const [grants, setGrants] = useState<{ id: number; text: string }[]>([]);

  const hasFetched = useRef(false);
  const [deleteModalOpen, setDeleteModalOpen] = useState(false);
  const [userModalOpen, setUserModalOpen] = useState(false);
  const [userModalMode, setUserModalMode] = useState<"add" | "remove">("add");

  useEffect(() => {
    const fetchGrants = async () => {
      try {
        const grantList = await readGrantList();
        setGrants(grantList);
      } catch (error) {
        console.error("Failed to fetch grants:", error);
      }
    };
    void fetchGrants();
  }, [readGrantList]);

  useEffect(() => {
    if (!hasFetched.current && numericId) {
      hasFetched.current = true;
      // Find group from store
      const groups = useUserGroupsStore.getState().userGroupList;
      const existingGroup = groups.find((g) => g.userGroupId === numericId);
      if (existingGroup) {
        setTimeout(() => {
          setFormData({
            groupName: existingGroup.groupName,
            userGrantList: existingGroup.userGrantList,
            updateTs: existingGroup.updateTs.toString(),
          });
        }, 0);
      }
      // Load users for the group management
      void retrieveUserList();
    }
  }, [numericId, retrieveUserList]);

  //============================^HOOOK LINE ^====================

  if (authState !== "YES") {
    return <div>Checking Auth...</div>;
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const data = {
      groupName: formData.groupName,
      userGrantList: formData.userGrantList,
      updateTs: Number(formData.updateTs),
    };

    try {
      if (isNew) {
        await createUserGroup({
          groupName: formData.groupName,
          userGrantList: formData.userGrantList,
        });
        notifications.show({
          title: "Success",
          message: "User group created successfully",
          color: "green",
          autoClose: 3000,
          withCloseButton: false,
        });
      } else {
        await updateUserGroup(Number(userGroupId), data);
        notifications.show({
          title: "Success",
          message: "User group updated successfully",
          color: "green",
          autoClose: 3000,
          withCloseButton: false,
        });
      }
      void navigate(`/${ROUTES.USER_GROUP_LIST}`);
    } catch (error) {
      console.error("Error saving user group:", error);
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
        await deleteUserGroup(numericId);
        notifications.show({
          title: "Success",
          message: "User group deleted successfully",
          color: "green",
          autoClose: 3000,
          withCloseButton: false,
        });
      void navigate(`/${ROUTES.USER_GROUP_LIST}`);
      } catch (error) {
        console.error("Error deleting user group:", error);
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
      id => userList.find(u => u.appUserId === id)?.updateTs || 0
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
      console.error(`Error ${userModalMode}ing users:`, error);
      notifications.show({
        title: "Error",
        message: `Failed to ${userModalMode} users`,
        color: "red",
        autoClose: 5000,
      });
    }
  };

  return (
    <div style={{ padding: "2rem" }}>
      <Title order={1} mb="lg">
        Edit User Group
      </Title>
      <form onSubmit={(e) => void handleSubmit(e)}>
        <TextInput
          label="Group Name"
          value={formData.groupName}
          onChange={(e) =>
            setFormData({ ...formData, groupName: e.target.value })
          }
          required
          mb="md"
        />
        <MultiSelect
          label="Roles"
          data={grants.map((g) => ({ value: g.id.toString(), label: g.text }))}
          value={formData.userGrantList.map(String)}
          onChange={(values) =>
            setFormData({ ...formData, userGrantList: values.map(Number) })
          }
          mb="md"
        />
        <Group>
          <Button type="submit">Save</Button>
          <Button color="red" onClick={handleDeleteClick}>
            Delete
          </Button>
          <Button
            variant="outline"
            onClick={() => void navigate(`/${ROUTES.USER_GROUP_LIST}`)}
          >
            Cancel
          </Button>
        </Group>
      </form>

      {/* User Management Section */}
      <div style={{ marginTop: "2rem" }}>
        <Title order={2} mb="md">
          Group Members
        </Title>
        {(() => {
          const groupMemberList = userList.filter((u) =>
            u.userGroupMembershipList.includes(numericId || 0),
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
      </div>

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
    </div>
  );
}

export default UserGroupEditPage;
