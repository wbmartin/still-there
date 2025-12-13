import { useState, useEffect, useRef } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import {
  TextInput,
  Button,
  Title,
  Group,
  Checkbox,
  MultiSelect,
} from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { useAppUserStore } from "../stores/app-user-store";
import { useUserGroupsStore } from "../stores/user-groups-store";
import useAuthGuard from "../hooks/use-auth-guard";
import { DeleteConfirmModal } from "../components/DeleteConfirmModal";
import { ROUTES } from "../constants/routes";
import type { UserFormData, User } from "../types/user-type";
import { defaultUserFormData } from "../types/user-type";

function UserEditPage() {
  const authState = useAuthGuard();
  const [searchParams] = useSearchParams();
  const { createUser, updateUser, deleteUser } = useAppUserStore();
  const { readUserGroupList, userGroups } = useUserGroupsStore();
  const navigate = useNavigate();

  const id = searchParams.get("appUserId") || "";
  const numericId = isNaN(Number(id)) ? null : Number(id);

  const [formData, setFormData] = useState<UserFormData>(defaultUserFormData);

  const isNew = id === "new";
  const [userLoading, setUserLoading] = useState(!isNew && numericId !== null);
  const hasFetched = useRef(false);
  const [deleteModalOpen, setDeleteModalOpen] = useState(false);

  useEffect(() => {
    if (!hasFetched.current) {
      hasFetched.current = true;
      readUserGroupList();
      if (numericId && !isNew) {
        // Find user from store
        const users = useAppUserStore.getState().users;
        const existingUser = users.find((u: User) => u.appUserId === numericId);
        if (existingUser) {
          setTimeout(() => {
            setFormData({
              userName: existingUser.userName,
              firstName: existingUser.firstName,
              lastName: existingUser.lastName,
              email: existingUser.email,
              userGroupMembershipList: existingUser.userGroupMembershipList,
              mfaExempt: existingUser.mfaExempt,
              pwHash: "",
              updateTs: existingUser.updateTs.toString(),
            });
            setUserLoading(false);
          }, 0);
        } else {
          setTimeout(() => {
            setUserLoading(false);
          }, 0);
        }
      }
    }
  }, [numericId, isNew, readUserGroupList]);

  //============================^HOOOK LINE ^====================

  if (authState !== "YES") {
    return <div>Checking Auth...</div>;
  }

  if (userLoading) {
    return <div>Loading user...</div>;
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const data = {
      userName: formData.userName,
      firstName: formData.firstName,
      lastName: formData.lastName,
      email: formData.email,
      userGroupMembershipList: formData.userGroupMembershipList,
      mfaExempt: formData.mfaExempt,
      pwHash: formData.pwHash || undefined,
      updateTs: !isNew ? Number(formData.updateTs) : undefined,
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
        await updateUser(Number(id), data);
        notifications.show({
          title: "Success",
          message: "User updated successfully",
          color: "green",
          autoClose: 3000,
          withCloseButton: false,
        });
      }
      navigate(`/${ROUTES.APP_USER_LIST}`);
    } catch (error) {
      console.error("Error saving user:", error);
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
    setDeleteModalOpen(true);
  };

  const handleDeleteConfirm = async () => {
    if (numericId) {
      try {
        await deleteUser(numericId);
        notifications.show({
          title: "Success",
          message: "User deleted successfully",
          color: "green",
          autoClose: 3000,
          withCloseButton: false,
        });
        navigate(`/${ROUTES.APP_USER_LIST}`);
      } catch (error) {
        console.error("Error deleting user:", error);
        notifications.show({
          title: "Error",
          message: "Failed to delete user",
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

  const groupOptions = userGroups.map((g) => ({
    value: g.userGroupId.toString(),
    label: g.groupName,
  }));

  return (
    <div style={{ padding: "2rem" }}>
      <Title order={1} mb="lg">
        {isNew ? "Add User" : "Edit User"}
      </Title>
      <form onSubmit={handleSubmit}>
        <TextInput
          label="Username"
          value={formData.userName}
          onChange={(e) =>
            setFormData({ ...formData, userName: e.target.value })
          }
          required
          mb="md"
        />
        <TextInput
          label="First Name"
          value={formData.firstName}
          onChange={(e) =>
            setFormData({ ...formData, firstName: e.target.value })
          }
          required
          mb="md"
        />
        <TextInput
          label="Last Name"
          value={formData.lastName}
          onChange={(e) =>
            setFormData({ ...formData, lastName: e.target.value })
          }
          required
          mb="md"
        />
        <TextInput
          label="Email"
          value={formData.email}
          onChange={(e) => setFormData({ ...formData, email: e.target.value })}
          required
          mb="md"
        />
        {isNew && (
          <TextInput
            label="Password"
            type="password"
            value={formData.pwHash}
            onChange={(e) =>
              setFormData({ ...formData, pwHash: e.target.value })
            }
            required
            mb="md"
          />
        )}
        <MultiSelect
          label="Groups"
          data={groupOptions}
          value={formData.userGroupMembershipList.map(String)}
          onChange={(values) =>
            setFormData({
              ...formData,
              userGroupMembershipList: values.map(Number),
            })
          }
          mb="md"
        />
        <Checkbox
          label="MFA Exempt"
          checked={formData.mfaExempt}
          onChange={(e) =>
            setFormData({ ...formData, mfaExempt: e.target.checked })
          }
          mb="md"
        />
        <Group>
          <Button type="submit">Save</Button>
          {!isNew && (
            <Button color="red" onClick={handleDeleteClick}>
              Delete
            </Button>
          )}
          <Button
            variant="outline"
            onClick={() => navigate(`/${ROUTES.APP_USER_LIST}`)}
          >
            Cancel
          </Button>
        </Group>
      </form>
      <DeleteConfirmModal
        isOpen={deleteModalOpen}
        onConfirm={handleDeleteConfirm}
        onCancel={handleDeleteCancel}
        entityType="user"
      />
    </div>
  );
}

export default UserEditPage;
